package rrsm

/*
 * configure and init options
 */

import (
	"fmt"
	"sync"
	"time"

	"github.com/hhyasdf/rrsm/rpc"
)

// reconfigure the nodes of rrsm, only for nodes
func (self *RRSMNode) ApplyConfig(nodes []string) error {
	if len(nodes) <= 0 {
		return fmt.Errorf("config err: amounts of nodes needed to be a positive number!")
	}
	self.config.Nodes = nodes

	// rebuild a node
	// ....

	return nil
}

func NewNode() *RRSMNode {
	return &RRSMNode{}
}

// init a rrsm node
// addr is the address of the node itself
// build the whole network between all the nodes
func (self *RRSMNode) Build(addr string, initState RRSMState, configuration *RRSMConfig,
	RPCListenPort string, electionTimeout time.Duration, heartbeatInterval time.Duration) error {
	self.InitState = initState
	self.CurrentState = initState
	self.nodes = make(map[string]*rpc.RPCClient)
	self.addr = addr

	// init timer
	self.electionTimeoutTicker = nil
	self.electionTimeout = electionTimeout
	self.heartbeatTimeTicker = nil
	self.heartbeatInterval = heartbeatInterval

	// become a follower at the beginning
	self.character = RaftFOLLOWER
	self.currentTerm = uint64(0)
	self.haveVoted = false

	// init channels
	self.newTermChan = make(chan int, 1)
	self.accessLeaderChan = make(chan int, 1)

	// init lock
	self.termChangingLock = &sync.Mutex{}
	self.leaderChangingLock = &sync.Mutex{}

	// init node configuration
	if configuration == nil {
		return fmt.Errorf("configuration is needed!")
	}
	if len(configuration.Nodes) <= 0 {
		return fmt.Errorf("config err: amounts of nodes needed to be a positive number!")
	}
	self.config = *configuration
	self.amountsOfNodes = uint32(len(self.config.Nodes))

	// register rpc service
	raftRPC := RaftRPC{
		node: self,
	}
	err := rpc.RPCRegister(RPCListenPort, &raftRPC)
	if err != nil {
		return err
	}

	// build rpc connection with other nodes
	for _, node := range self.config.Nodes {
		if node != addr {
			client, err := rpc.RPCConnect(node)
			if err != nil {
				// need to connect with all the nodes at the period of building
				return err
			} else {
				self.nodes[node] = client
			}
		}
	}

	return nil
}
