package main

import (
	"flag"
	"strings"
	"time"
	"math/rand"
)

func main() {
	// 1. Parse Command Line Arguments
	id := flag.String("id", "localhost:8001", "My ID (Address)")
	peersStr := flag.String("peers", "localhost:8002,localhost:8003", "Comma-separated peers")
	flag.Parse()

	peers := strings.Split(*peersStr, ",")

	// Seed random for election timers
	rand.Seed(time.Now().UnixNano())

	// 2. Start Raft RPC Server (This listens on localhost:8001)
	raft := NewRaftNode(*id, peers)
	raft.StartRPC() // <--- NEW: Open the phone lines!
	go raft.Run()

    // 2. Start DB Server (On a different port, e.g., 9001)
    // Extract port 8001 -> 9001
    // Simple hack for now:
    // This is just to keep the DB alive, we won't use it for this test yet.
    _ = strings.Split(*id, ":")[1]
    
    // Just block forever here so the program doesn't exit
    select {} 
}