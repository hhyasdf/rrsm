package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/hhyasdf/rrsm"
)

type state int

func (self *state) Info() string {
	return fmt.Sprintf("%d", self)
}

func (self *state) Handle(command string) interface{} {
	return nil
}

func main() {
	var initState state = 1
	config := rrsm.RRSMConfig{
		[]string{
			":50001",
			":50002",
			":50003",
			":50004",
			":50005",
			":50006",
			":50007",
			":50008",
			":50009",
			":50010",
		},
	}

	nodes := [10]*rrsm.RRSMNode{}
	for index, name := range config.Nodes {
		nodes[index] = rrsm.NewNode()
		err := nodes[index].Build(name, &initState, &config, name[1:],
			2*time.Second,
			1*time.Second,
		)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	for _, node := range nodes {
		go func(n *rrsm.RRSMNode) {
			runtime.LockOSThread()
			n.Run()
		}(node)
	}

	for {
		time.Sleep(5 * time.Second)
		fmt.Println("============")
		for _, node := range nodes {
			leader := node.GetLeaderAddr()
			if leader == "" {
				fmt.Println("no leader exist")
			} else {
				if node.GetCharacter() == rrsm.RaftLEADER {
					fmt.Println("I'm the leader")
				} else {
					fmt.Println(leader)
				}
			}
		}
		fmt.Println("============")
	}
}
