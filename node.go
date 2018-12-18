package rrsm

/*
 * main operations of the replicated state machine
 */

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/hhyasdf/rrsm/rpc"
)

const (
	RaftFOLLOWER  int = 1
	RaftCANDIDATE int = 2
	RaftLEADER    int = 3
)

type RRSMNode struct {
	config            RRSMConfig
	InitState         RRSMState
	nodes             map[string]*rpc.RPCClient
	amountsOfNodes    uint32
	addr              string
	CurrentState      RRSMState
	electionTimeout   time.Duration
	heartbeatInterval time.Duration

	// election time out and heartbeat interval
	electionTimeoutTicker *time.Ticker
	heartbeatTimeTicker   *time.Ticker

	// will be changed while working
	leaderAddr  string
	character   int
	currentTerm uint64
	haveVoted   bool

	// lock
	termChangingLock   *sync.Mutex
	leaderChangingLock *sync.Mutex

	// some event chan
	newTermChan      chan int // discovered a new term form others
	accessLeaderChan chan int
}

// change to a newer term
// will occur a new term event
func (self *RRSMNode) ChangetoANewTerm(newTernNumber uint64) {
	self.termChangingLock.Lock()
	defer self.termChangingLock.Unlock()

	self.currentTerm = newTernNumber
	self.haveVoted = false
	// try to send a new term event
	select {
	case self.newTermChan <- 1:
	default:
	}
}

// increase the current term number
// clean the voted flag
func (self *RRSMNode) CreateANewTerm() {
	self.termChangingLock.Lock()
	defer self.termChangingLock.Unlock()

	self.currentTerm++
	self.haveVoted = false
}

func (self *RRSMNode) GetCurrentTerm() uint64 {
	self.termChangingLock.Lock()
	defer self.termChangingLock.Unlock()

	termNum := self.currentTerm
	return termNum
}

func (self *RRSMNode) SetVotedFlag() {
	self.termChangingLock.Lock()
	defer self.termChangingLock.Unlock()

	self.haveVoted = true
}

func (self *RRSMNode) SetVotedFlagIfNot() bool {
	self.termChangingLock.Lock()
	defer self.termChangingLock.Unlock()

	if !self.haveVoted {
		self.haveVoted = true
		return true
	}
	return false
}

func (self *RRSMNode) SetLeaderAddr(addr string) {
	self.leaderChangingLock.Lock()
	defer self.leaderChangingLock.Unlock()

	self.leaderAddr = addr
}

func (self *RRSMNode) GetLeaderAddr() string {
	self.leaderChangingLock.Lock()
	defer self.leaderChangingLock.Unlock()

	return self.leaderAddr
}

func (self *RRSMNode) GetAddr() string {
	return self.addr
}

func (self *RRSMNode) GetCharacter() int {
	return self.character
}

// parallel requests to request vote
// return true if it becomes a leader,
// else if election timeout,
// or recieve a heartbeat from a new leader,
// or a new term appear, return false
func (self *RRSMNode) RequestVote() (success bool, err error) {
	// start a new term
	// empty the votedFlag
	self.CreateANewTerm()
	self.character = RaftCANDIDATE

	// vote for itself
	// if cannot vote for itself, it has already voted for somebody
	voteForMyself := self.SetVotedFlagIfNot()
	if !voteForMyself {
		return false, nil
	}
	var numberOfVotes uint32 = uint32(1)

	// init date structure used to elect
	var electedFlag int32 = 0
	electedChan := make(chan int, 1)

	self.resetElectionTimeoutTicker()

	// if it's the only node
	if len(self.config.Nodes) == 1 && self.config.Nodes[0] == self.addr {
		select {
		case electedChan <- 1:
		default:
		}
	}

	for _, c := range self.nodes {
		go func(client *rpc.RPCClient) {
			requestVoteArgs := &RequestVoteArgs{
				TermNumber: self.GetCurrentTerm(),
				Elector:    self.addr,
			}
			getVote := false
			for {
				err := rpc.RPCCall(client, "RaftRPC.RequestVote", requestVoteArgs, &getVote)
				if err == nil {
					break
				}
			}
			if getVote {
				// account the amounts of votes
				atomic.AddUint32(&numberOfVotes, 1)
				// if majority of nodes have voted for this node, it becomes a leader
				if numberOfVotes > (self.amountsOfNodes)/2 && electedFlag == 0 {
					atomic.StoreInt32(&electedFlag, 1)
					// to make sure every goroutine can be finished
					select {
					case electedChan <- 1:
					default:
					}
				}
			}
		}(c)
	}

	select {
	case <-electedChan:
		// become a leader
		self.character = RaftLEADER
		self.SetLeaderAddr(self.addr)
		return true, nil
	case <-self.electionTimeoutTicker.C:
		// this should be a split vote situation
		// need to restart an election
	case <-self.accessLeaderChan:
		// a new leader appears
	case <-self.newTermChan:
		// if a new term event happend before a election, and without empty channel
		// it may cause a split vote, but it doesn't matter
		// new term appears
	}

	self.character = RaftFOLLOWER
	return false, nil
}

// send heartbeat to followers as a leader
func (self *RRSMNode) SendHeartbeats() error {
	for _, c := range self.nodes {
		go func(client *rpc.RPCClient) {
			appendEntriesArgs := AppendEntriesArgs{
				LeaderAddr:  self.addr,
				IsHeartbeat: true,
				TermNumber:  self.GetCurrentTerm(),
			}
			for {
				err := rpc.RPCCall(client, "RaftRPC.AppendEntries", &appendEntriesArgs, nil)
				if err == nil {
					break
				}
			}
		}(c)
	}
	return nil
}

// try to commit an log entry as a leader
func (self *RRSMNode) AppendEntriesArgs(command string) (commited bool, err error) {
	return true, nil
}

func (self *RRSMNode) WorkAsLeader() error {
	self.stopElectionTimeoutTicker()
	self.startHeartbeatTicker()

LeaderLoop:
	for {
		select {
		case <-self.heartbeatTimeTicker.C:
			// send heartbeats
			self.SendHeartbeats()
		case <-self.newTermChan:
			// only a new term appears
			// will a leader becomes a follower
			break LeaderLoop
		}
	}

	// handle commands
	// ...

	self.startElectionTimeoutTicker()
	self.stopHeartbeatTicker()
	self.character = RaftFOLLOWER
	return nil
}

func (self *RRSMNode) Run() error {
	self.startElectionTimeoutTicker()
	for {
		// start to work as a follower
	FollowerLoop:
		for {
			select {
			case <-self.accessLeaderChan:
			case <-self.newTermChan:
			case <-self.electionTimeoutTicker.C:
				{
					break FollowerLoop
				}
			}
			self.resetElectionTimeoutTicker()
		}

		// become a candidate
		if win, _ := self.RequestVote(); win {
			// become a leader
			self.WorkAsLeader()
		}
	}
}
