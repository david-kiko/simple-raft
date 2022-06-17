package raft

func (rf *Raft) Store(data string) error {
	rf.log = append(rf.log, LogEntry{rf.currentTerm, rf.commitIndex + 1, data})
	return nil
}
