package main

import (
	"flag"
	"github.com/phuslu/log"
	"os"
	"simple-raft/raft"
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

func getNodes(clusters map[int]string, id int) (result map[int]string) {
	result = make(map[int]string)
	for k, v := range clusters {
		if k == id {
			continue
		}
		result[k] = v
	}
	return result
}

func main() {
	port := flag.String("port", ":9091", "rpc listen port")
	id := flag.Int("id", 1, "node ID")
	flag.Parse()

	clusters := map[int]string{
		0: "127.0.0.1:12379",
		1: "127.0.0.1:22379",
		2: "127.0.0.1:32379",
		3: "127.0.0.1:42379",
		4: "127.0.0.1:52379",
	}

	raftIns := &raft.Raft{}
	raftIns.Me = *id
	raftIns.Nodes = getNodes(clusters, *id)
	raftIns.Rpc(*port)
	raftIns.Start()

	select {}

}
