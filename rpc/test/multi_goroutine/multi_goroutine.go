package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hhyasdf/rrsm/rpc"
)

type Service struct {
	counter int64
}

// self will be regarded as a input parameter
// so the input parameter number is 3...
func (self *Service) Add(args *int64, reply *int64) error {
	atomic.AddInt64(&self.counter, *args)
	*reply = self.counter
	return nil
}

func (self *Service) Read(args int64, reply *int64) error {
	*reply = atomic.LoadInt64(&self.counter)
	return nil
}

func main() {
	server := rpc.NewServer()
	err := server.Register(&Service{0})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	go server.Serve("40000")

	client := rpc.NewClient()
	client.Connect(":40000")

	var counter int64 = 10013452
	var a int64 = 1
	for i := 0; i < 5; i++ {
		go func(c *rpc.RPCClient) {
			for {
				err := c.Call("Service.Add", &a, &counter)
				if err == nil {
					fmt.Println(counter)
				}
			}
		}(client)
	}
	for {
		time.Sleep(1000) // the schedule module of go need to be blocked
	}
}
