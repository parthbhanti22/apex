package main

import (
	"net"
	"net/rpc"
	"log"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

//1.Vote Request (candidate -> follower)
type RequestVoteArgs struct {
	Term int
	CandidateID string
}

type RequestVoteReply struct {
	Term int
	VoteGranted bool
}
//2.HeartBeat/ Log Replication (Leader -> Follower)
type AppendEntriesArgs struct {
	Term int
	LeaderID string
	//We will add the log entries later
}
//3.AppendEntries (Follower -> Leader)
type AppendEntriesReply struct {
	Term int
	Success bool
}
// The 3 raft states

const(
	Follower = iota
	Candidate
	Leader
)

type RaftNode struct {
	mu sync.Mutex //lock for thread safety
	
	id string //My id(eg. "localhost:8001")
	peers []string //list of other nodes (eg. on local host 8002,8003)

	state int //am i follower, candidate or leader
	currentTerm int //The "era" we are in (increments on every election)
	votedFor string //Who did I vote for in the current term

	//heartbeat channels
	electionResetEvent time.Time
	
	peerClients map[string]*rpc.Client
}

//StartRPC starts the internal Raft server
func (rn *RaftNode) StartRPC() {
	server := rpc.NewServer() 
	server.Register(rn) //register our raftNode methods

	//Listen on the port defiend in -id (eg. localhost:8001)
	listener, err := net.Listen("tcp",rn.id)
	if err != nil {
		log.Fatalf("Raft RPC Listen error: %v",err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			//serve connection in a new goroutine
			go server.ServeConn(conn)
		}
	}()
}

// RequestVote is called by a Candidate to ask for a vote
func (rn *RaftNode) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	// 1. If the candidate is outdated (older term), reject.
	if args.Term < rn.currentTerm {
		reply.Term = rn.currentTerm
		reply.VoteGranted = false
		return nil
	}

	// 2. If we see a newer term, we must step down to Follower
	if args.Term > rn.currentTerm {
		rn.currentTerm = args.Term
		rn.state = Follower
		rn.votedFor = ""
	}

	// 3. Vote! (If we haven't voted yet)
	if rn.votedFor == "" || rn.votedFor == args.CandidateID {
		rn.votedFor = args.CandidateID
		rn.electionResetEvent = time.Now() // Reset timeout (Leader is alive-ish)
		reply.VoteGranted = true
		fmt.Printf("[%s] ✅ Voted for %s (Term %d)\n", rn.id, args.CandidateID, rn.currentTerm)
	} else {
		reply.VoteGranted = false
	}
	
	reply.Term = rn.currentTerm
	return nil
}

// AppendEntries is called by the Leader to sync logs (and heartbeat)
func (rn *RaftNode) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	// 1. Reject outdated leaders
	if args.Term < rn.currentTerm {
		reply.Term = rn.currentTerm
		reply.Success = false
		return nil
	}

	// 2. If we see a valid leader, update our state
	if args.Term >= rn.currentTerm {
		rn.currentTerm = args.Term
		rn.state = Follower // Ensure we are follower
		rn.electionResetEvent = time.Now() // RESET TIMER! We heard from boss.
	}

	reply.Term = rn.currentTerm
	reply.Success = true
	return nil
}


func (rn *RaftNode) getPeerClient(peerID string) (*rpc.Client, error) {
	rn.mu.Lock()
	client, ok := rn.peerClients[peerID]
	rn.mu.Unlock()

	if ok && client != nil {
		return client, nil
	}

	// Dial if we don't have a connection
	newClient, err := rpc.Dial("tcp", peerID)
	if err != nil {
		return nil, err
	}

	rn.mu.Lock()
	rn.peerClients[peerID] = newClient
	rn.mu.Unlock()
	
	return newClient, nil
}

func NewRaftNode(id string, peers []string) *RaftNode {
	return &RaftNode{
		id:                 id,
		peers:              peers,
		state:              Follower,
		electionResetEvent: time.Now(),
		peerClients:        make(map[string]*rpc.Client),
	}
}

//Run starts the main loop (the heartbeat monitor)
func(rn *RaftNode) Run() {
	//loop forever
	for {
		//check state and decide what to do
		switch rn.state {
		case Follower:
			rn.runFollower()
		case Candidate:
			rn.runCandidate()
		case Leader:
			rn.runLeader()
		}
	}
}

func (rn *RaftNode) runFollower() {
	// Check if we need to timeout
	rn.mu.Lock()
	lastHeartbeat := rn.electionResetEvent
	rn.mu.Unlock()

	// If too much time passed since last heartbeat
	if time.Since(lastHeartbeat) > time.Duration(150+rand.Intn(150))*time.Millisecond {
		fmt.Printf("[%s] ⏰ Election Timeout! No heartbeat.\n", rn.id)
		rn.mu.Lock()
		rn.state = Candidate
		rn.mu.Unlock()
	}
	
	time.Sleep(20 * time.Millisecond) // Tick quickly
}

func (rn *RaftNode) runCandidate() {
	rn.mu.Lock()
	rn.currentTerm++
	rn.votedFor = rn.id
	rn.state = Candidate // Ensure state
	term := rn.currentTerm
	rn.mu.Unlock()

	fmt.Printf("[%s] 🗳️ Campaigning for Term %d...\n", rn.id, term)

	// Counter for votes (Starts with 1 vote: myself)
	votesReceived := 1
	votesNeeded := (len(rn.peers) + 1) / 2 + 1 

	// Send RequestVote to all peers
	for _, peer := range rn.peers {
		go func(peer string) {
			args := RequestVoteArgs{
				Term:        term,
				CandidateID: rn.id,
			}
			var reply RequestVoteReply

			// Dial the peer
			client, err := rn.getPeerClient(peer)
			if err != nil {
				return // Peer is probably down
			}

			// Make the RPC call
			err = client.Call("RaftNode.RequestVote", &args, &reply)
			if err == nil {
				rn.mu.Lock()
				defer rn.mu.Unlock()

				// Check if we got a vote
				if reply.VoteGranted && rn.state == Candidate {
					votesReceived++
					if votesReceived >= votesNeeded {
						// WE WON!
						fmt.Printf("[%s] 🎉 I WON THE ELECTION! Becoming Leader.\n", rn.id)
						rn.state = Leader
						return
					}
				} else if reply.Term > rn.currentTerm {
					// Opps, someone else has a higher term. Step down.
					rn.currentTerm = reply.Term
					rn.state = Follower
				}
			}
		}(peer)
	}

	// Wait for election timeout to re-run loop
	time.Sleep(time.Duration(150+rand.Intn(150)) * time.Millisecond)
}

func (rn *RaftNode) runLeader() {
	// Send Heartbeats immediately
	rn.broadcastHeartbeat()
	
	time.Sleep(50 * time.Millisecond)
}

func (rn *RaftNode) broadcastHeartbeat() {
	rn.mu.Lock()
	term := rn.currentTerm
	id := rn.id
	rn.mu.Unlock()

	for _, peer := range rn.peers {
		go func(peer string) {
			args := AppendEntriesArgs{
				Term:     term,
				LeaderID: id,
			}
			var reply AppendEntriesReply

			client, err := rn.getPeerClient(peer)
			if err != nil { return }

			err = client.Call("RaftNode.AppendEntries", &args, &reply)
			if err != nil {
				rn.mu.Lock()
				rn.peerClients[peer] = nil
				rn.mu.Unlock()
			}
		}(peer)
	}
}