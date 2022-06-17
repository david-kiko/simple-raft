package raft

import (
	"github.com/phuslu/log"
	"math/rand"
	"net/http"
	"net/rpc"
	"time"
)

// State def
type State int

// status of node
const (
	Follower State = iota + 1
	Candidate
	Leader
)

func (c State) String() string {
	switch c {
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	case Leader:
		return "leader"
	default:
		return ""
	}
}

// LogEntry struct
type LogEntry struct {
	LogTerm  int
	LogIndex int
	Data     string
}

// Raft Node
type Raft struct {
	Me          int
	Nodes       map[int]string
	state       State
	currentTerm int
	votedFor    int
	voteCount   int
	voteList    []int      //拥有的选票
	log         []LogEntry // 日志条目集合
	commitIndex int        // 被提交的最大索引
	lastApplied int        // 被应用到状态机的最大索引
	nextIndex   []int      // 保存需要发送给每个节点的下一个条目索引
	matchIndex  []int      // 保存已经复制给每个节点日志的最高索引
	heartbeatC  chan bool
	toLeaderC   chan bool
}

// RequestVote rpc method
func (rf *Raft) RequestVote(args VoteArgs, reply *VoteReply) error {

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return nil
	}

	if rf.votedFor == -1 {
		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateID
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
	}

	return nil
}

// Heartbeat rpc method
func (rf *Raft) Heartbeat(args HeartbeatArgs, reply *HeartbeatReply) error {

	// 如果 leader 节点小于当前节点 term
	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return nil
	}

	// 如果只是 heartbeat
	rf.heartbeatC <- true
	if len(args.Entries) == 0 {
		reply.Success = true
		reply.Term = rf.currentTerm
		return nil
	}

	// 如果有 entries
	// leader 维护的 LogIndex 大于当前 Follower 的 LogIndex
	// 代表当前 Follower 失联过，所以 Follower 要告知 Leader 它当前
	// 的最大索引，以便下次心跳 Leader 返回
	if args.PrevLogIndex > rf.getLastIndex() {
		reply.Success = false
		reply.Term = rf.currentTerm
		reply.NextIndex = rf.getLastIndex() + 1
		return nil
	}

	rf.log = append(rf.log, args.Entries...)
	rf.commitIndex = rf.getLastIndex()
	reply.Success = true
	reply.Term = rf.currentTerm
	reply.NextIndex = rf.getLastIndex() + 1

	return nil
}

func (rf *Raft) Rpc(port string) {
	rpc.Register(rf)
	rpc.HandleHTTP()
	go func() {
		err := http.ListenAndServe(port, nil)
		if err != nil {
			log.Fatal().Msgf("listen error: ", err)
		}
	}()
}

func (rf *Raft) Start() {
	rf.state = Follower
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.heartbeatC = make(chan bool)
	rf.toLeaderC = make(chan bool)

	go func() {
		rand.Seed(time.Now().UnixNano())

		for {
			switch rf.state {
			case Follower:
				timeout := time.Duration(rand.Intn(5-3)+3) * time.Second

				select {
				case <-rf.heartbeatC:
					ws(rf, timeout, "update")
				case <-time.After(timeout):
					rf.state = Candidate
					ws(rf, timeout, "update")
				}
			case Candidate:
				rf.currentTerm++
				rf.votedFor = rf.Me
				rf.voteList = []int{rf.Me}
				rf.voteCount = 1
				go rf.broadcastRequestVote()

				timeout := time.Duration(rand.Intn(5-3)+3) * time.Second
				select {
				case <-time.After(timeout):
					rf.state = Follower
					ws(rf, timeout, "update")
				case <-rf.toLeaderC:
					rf.state = Leader
					ws(rf, timeout, "update")

					// 初始化 peers 的 nextIndex 和 matchIndex
					rf.nextIndex = make([]int, len(rf.Nodes)+1)
					rf.matchIndex = make([]int, len(rf.Nodes)+1)
					for i := range rf.Nodes {
						rf.nextIndex[i] = 1
						rf.matchIndex[i] = 0
					}
				}
			case Leader:
				rf.broadcastHeartbeat()
				timeoutHeartbeat := time.Duration(2) * time.Second
				ws(rf, timeoutHeartbeat, "heartbeat")
				time.Sleep(timeoutHeartbeat)
			}
		}

	}()
}

type VoteArgs struct {
	Term        int
	CandidateID int
}

type VoteReply struct {
	Term        int  //当前任期号， 以便候选人去更新自己的任期号
	VoteGranted bool //候选人赢得此张选票时为真
}

func (rf *Raft) broadcastRequestVote() {

	for otherID, addr := range rf.Nodes {
		go rf.sendRequestVote(otherID, addr)
	}
}

func (rf *Raft) sendRequestVote(otherID int, addr string) {
	args := VoteArgs{
		Term:        rf.currentTerm,
		CandidateID: rf.Me,
	}
	var reply VoteReply
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Error().Msgf("连接 %s 的rpc服务错误: %s", rf.Nodes[otherID], err)
		return
	}

	defer client.Close()
	client.Call("Raft.RequestVote", args, &reply)

	// 当前candidate节点无效
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.state = Follower
		rf.votedFor = -1
		timeout := time.Duration(rand.Intn(5-3)+3) * time.Second
		ws(rf, timeout, "update")
		return
	}

	if reply.VoteGranted {
		rf.voteCount++
		rf.voteList = append(rf.voteList, otherID)
	}

	if rf.voteCount >= len(rf.Nodes)/2+1 {
		rf.toLeaderC <- true
	}

}

type HeartbeatArgs struct {
	Term         int
	LeaderID     int
	PrevLogIndex int        // 新日志之前的索引
	PrevLogTerm  int        // PrevLogIndex 的任期号
	Entries      []LogEntry // 准备存储的日志条目（表示心跳时为空）
	LeaderCommit int        // Leader 已经commit的索引值
}

type HeartbeatReply struct {
	Success   bool
	Term      int
	NextIndex int // 如果 Follower Index小于 Leader Index， 会告诉 Leader 下次开始发送的索引位置
}

func (rf *Raft) broadcastHeartbeat() {
	for otherID, addr := range rf.Nodes {

		var args HeartbeatArgs
		args.Term = rf.currentTerm
		args.LeaderID = rf.Me
		args.LeaderCommit = rf.commitIndex

		// 计算 preLogIndex 、preLogTerm
		// 提取 preLogIndex - baseIndex 之后的entry，发送给 follower
		prevLogIndex := rf.nextIndex[otherID] - 1
		if rf.getLastIndex() > prevLogIndex {
			args.PrevLogIndex = prevLogIndex
			args.PrevLogTerm = rf.log[prevLogIndex].LogTerm
			args.Entries = rf.log[prevLogIndex:]
			log.Info().Msgf("send entries: %v\n", args.Entries)
		}

		go rf.sendHeartbeat(otherID, addr, args)
	}
}

func (rf *Raft) sendHeartbeat(otherID int, addr string, args HeartbeatArgs) {
	var reply HeartbeatReply
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Error().Msgf("连接 %s 的rpc服务错误: %s", addr, err)
		return
	}

	defer client.Close()
	client.Call("Raft.Heartbeat", args, &reply)

	// 如果 leader 节点落后于 follower 节点
	if reply.Success {
		if reply.NextIndex > 0 {
			rf.nextIndex[otherID] = reply.NextIndex
			rf.matchIndex[otherID] = rf.nextIndex[otherID] - 1
		}
	} else {
		// 如果 leader 的 term 小于 follower 的 term， 需要将 leader 转变为 follower 重新选举
		if reply.Term > rf.currentTerm {
			rf.currentTerm = reply.Term
			rf.state = Follower
			rf.votedFor = -1
			timeout := time.Duration(rand.Intn(5-3)+3) * time.Second
			ws(rf, timeout, "update")
			return
		}
	}
}

func (rf *Raft) getLastIndex() int {
	rlen := len(rf.log)
	if rlen == 0 {
		return 0
	}
	return rf.log[rlen-1].LogIndex
}

func (rf *Raft) getLastTerm() int {
	rlen := len(rf.log)
	if rlen == 0 {
		return 0
	}
	return rf.log[rlen-1].LogTerm
}
