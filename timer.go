package rrsm

import (
	"time"
)

func (self *RRSMNode) resetElectionTimeoutTicker() {
	self.stopElectionTimeoutTicker()
	self.startElectionTimeoutTicker()
}

func (self *RRSMNode) startElectionTimeoutTicker() {
	self.electionTimeoutTicker = time.NewTicker(self.electionTimeout)
}

func (self *RRSMNode) stopElectionTimeoutTicker() {
	if self.electionTimeoutTicker != nil {
		ticker := self.electionTimeoutTicker
		self.electionTimeoutTicker = nil
		ticker.Stop()
	}
}

func (self *RRSMNode) startHeartbeatTicker() {
	self.heartbeatTimeTicker = time.NewTicker(self.heartbeatInterval)
}

func (self *RRSMNode) stopHeartbeatTicker() {
	if self.heartbeatTimeTicker != nil {
		ticker := self.heartbeatTimeTicker
		self.heartbeatTimeTicker = nil
		ticker.Stop()
	}
}
