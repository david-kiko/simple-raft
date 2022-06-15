package main

import (
	"flag"
	"github.com/phuslu/log"
	"os"
	"simple-raft/raft"
	"strings"
)

func init() {
	if !log.IsTerminal(os.Stderr.Fd()) {
		return
	}
	log.DefaultLogger = log.Logger{
		TimeFormat: "15:04:05",
		Caller:     1,
		Writer: &log.ConsoleWriter{
			ColorOutput:    true,
			QuoteString:    true,
			EndWithMessage: true,
		},
	}
}

func main() {
	port := flag.String("port", ":9091", "rpc listen port")
	cluster := flag.String("cluster", "127.0.0.1:9091", "comma sep")
	id := flag.Int("id", 1, "node ID")

	flag.Parse()
	clusters := strings.Split(*cluster, ",")

	ns := make(map[int]*raft.Node)
	for k, v := range clusters {
		ns[k] = raft.NewNode(v)
	}

	raftIns := &raft.Raft{}
	raftIns.Me = *id
	raftIns.Nodes = ns
	raftIns.Rpc(*port)
	raftIns.Start()

	select {}

}
