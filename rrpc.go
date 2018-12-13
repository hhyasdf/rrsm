package rrsm

import (
	"fmt"
)

/*
 * implements of raft rpc
 */

type RaftRPC struct {
	node *RRSMNode
}

type AppendEntriesArgs struct {
	LeaderAddr    string
	TermNumber    uint64
	Command       string
	IsHeartbeat   bool
	IsConfigEntry bool
}

type RequestVoteArgs struct {
	TermNumber uint64
	Elector    string
}

// RequestVoteRPC of Raft
func (self *RaftRPC) RequestVote(requestVoteArgs RequestVoteArgs, vote *bool) error {
	// ignore the old term number
	// update current term if a new term appears
	if !self.node.freshTermCheck(requestVoteArgs.TermNumber) {
		return fmt.Errorf("this term number is not fresh!")
	}

	// if a node already voted then it will not vote again in the same term
	*vote = self.node.SetVotedFlagIfNot()

	return nil
}

// AppendEntriesRPC of Raft
func (self *RaftRPC) AppendEntries(appendEntriesArgs *AppendEntriesArgs, appended *bool) error {
	if appendEntriesArgs == nil {
		return fmt.Errorf("appendEntriesArgs should not be nil!")
	}

	// ignore the old term number
	// update current term if a new term appears
	if !self.node.freshTermCheck(appendEntriesArgs.TermNumber) {
		return fmt.Errorf("this term number is not fresh!")
	}

	// recognize heartbeat
	if appendEntriesArgs.IsHeartbeat {
		// reset the election timeout
		self.node.resetElectionTimeoutTicker()
		// set the leader address
		self.node.SetLeaderAddr(appendEntriesArgs.LeaderAddr)
		// try to send a access leader event
		select {
		case self.node.accessLeaderChan <- 1:
		default:
		}
		*appended = true
	} else {
		if appendEntriesArgs.IsConfigEntry {
			// append configuration entry
		} else {
			// append normal log entries
		}
	}

	return nil
}

// try to update current term number if a new term appears,
// if a old term number comes, return false, else return true
func (self *RRSMNode) freshTermCheck(termNumbr uint64) bool {
	currentTerm := self.GetCurrentTerm()
	if currentTerm < termNumbr {
		// update term number
		// means that the follower can vote again, leader or candidate need to change to follower
		self.ChangetoANewTerm(termNumbr)

		// will not reset election timeout here
	} else if currentTerm > termNumbr {
		// ignore the request of stale term number
		return false
	}
	return true
}
