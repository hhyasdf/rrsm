package rpc

import (
	"fmt"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
)

type RPCClient struct {
	conn       *RPCConn
	serverAddr string

	// the strings in this chan is address need to reconnect
	reConnectChan chan int

	// make sure the parallel RPC request while be handled in order
	sequentialMutex *sync.Mutex

	// every RPC request for the same connection has a unique serial number
	// it will not be zero
	serialNumber uint64
}

type RPCConn struct {
	c net.Conn
}

func NewClient() *RPCClient {
	client := &RPCClient{}
	client.reConnectChan = make(chan int, 1)
	client.sequentialMutex = &sync.Mutex{}
	client.serialNumber = 1
	client.conn = nil
	return client
}

func (self *RPCClient) Connect(address string) error {
	// start a goroutine for each tcp connection to keep connecting
	self.serverAddr = address
	return nil
}

// don't return the RPC error but return the error of network or function
// if send seccuss, return nil
func (self *RPCClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	t := reflect.TypeOf(args)
	v := reflect.ValueOf(args)

	// if args is a pointer, transmit the value it points to !!
	parameters := args
	if t.Kind() == reflect.Ptr {
		parameters = v.Elem().Interface()
	}

	self.sequentialMutex.Lock()
	defer self.sequentialMutex.Unlock()

	if self.conn == nil {
		conn, err := net.Dial("tcp", self.serverAddr)
		if err != nil {
			return NetworkError{}
		} else {
			self.conn = &RPCConn{c: conn}
		}
	}

	r, err := SendRPCRequest(self.serialNumber, serviceMethod, parameters, self.conn.c)
	if err != nil {
		if reflect.TypeOf(err) == reflect.TypeOf(NetworkError{}) {
			// maybe network problem happens
			// will try to reconnect in next call
			atomic.StoreUint64(&self.serialNumber, 0)
			self.conn.c.Close()
			self.conn = nil
		}
		return err
	}

	// increase the serial number, should be wrap-around
	if !atomic.CompareAndSwapUint64(&self.serialNumber, ^uint64(0), 1) {
		atomic.AddUint64(&self.serialNumber, 1)
	}

	// set the object that reply point to with r
	// if reply == nil, means the caller don't need the return value
	if reply != nil {
		p := reflect.ValueOf(reply).Elem()
		if p.CanSet() {
			p.Set(reflect.ValueOf(r))
		} else {
			return fmt.Errorf("reply should be a pointer which can be overwrite")
		}
	}

	return nil
}

// send a RPC request and wait for response
// make sure a broken packet will be resent
func SendRPCRequest(requestSerialNumber uint64, method string, args interface{},
	conn net.Conn) (reply interface{}, err error) {
	for {
		err = SendRPCPacket(conn, requestSerialNumber, method, args, nil)
		if err != nil {
			return nil, err
		}

		_, serviceMethod, _, r, err := RecieveRPCPacket(conn)
		if err != nil {
			// the packet recieved is broken, resend the RPC request
			if reflect.TypeOf(err) != reflect.TypeOf(BadPacketError{}) {
				continue
			}
			return nil, err
		}

		// the packet sended is broken, recieve a useless response from the other side
		// resend the broken packet
		if serviceMethod != "" {
			reply = r
			break
		}
	}

	return reply, nil
}
