# simple implementation of raft consensus algorithm

---

### TODO
- [x] Leader election
- [x] Log replication

### Branch
- master： Leader election & Log replication
- election：Leader election
- replication：Log replication

### build
```
go build -o simple-raft
```

### run
```
go install github.com/mattn/goreman@latest
goreman start
```

