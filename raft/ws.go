package raft

import (
	"bytes"
	"encoding/json"
	"github.com/phuslu/log"
	"net/http"
	"time"
)

type WsMessage struct {
	Event string `json:"event"`
	Msg   struct {
		ID          int        `json:"id"`
		State       string     `json:"state"`
		Term        int        `json:"term"`
		Timeout     int        `json:"timeout"`
		VotedFor    int        `json:"voted_for"`
		VoteCount   int        `json:"voteCount"`
		VoteList    []int      `json:"voteList"`
		Log         []LogEntry `json:"log"`         // 日志条目集合
		CommitIndex int        `json:"commitIndex"` // 被提交的最大索引
		LastApplied int        `json:"lastApplied"` // 被应用到状态机的最大索引
		NextIndex   []int      `json:"nextIndex"`   // 保存需要发送给每个节点的下一个条目索引
		MatchIndex  []int      `json:"matchIndex"`  // 保存已经复制给每个节点日志的最高索引
	} `json:"msg"`
}

func httpPostJson(msg WsMessage) {

	payload, _ := json.Marshal(msg)
	url := "http://127.0.0.1:8000/send"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("http发送失败")
		return
	}
	resp.Body.Close()
}

func ws(rf *Raft, timeout time.Duration, event string) {
	data := WsMessage{
		Event: event,
	}

	data.Msg.ID = rf.Me
	data.Msg.Timeout = int(timeout.Milliseconds())
	data.Msg.Term = rf.currentTerm
	data.Msg.State = rf.state.String()
	data.Msg.VotedFor = rf.votedFor
	data.Msg.VoteCount = rf.voteCount
	data.Msg.VoteList = rf.voteList
	//data.Msg.Log = rf.log
	data.Msg.CommitIndex = rf.commitIndex
	data.Msg.LastApplied = rf.lastApplied
	data.Msg.NextIndex = rf.nextIndex
	data.Msg.MatchIndex = rf.matchIndex

	httpPostJson(data)
}
