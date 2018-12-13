package rpc

import (
	"fmt"
	"net"
	"reflect"
	"strings"
)

// one RPCSever listen to one port
type RPCSever struct {
	serviceMap map[string]interface{}
	listener   *net.Listener
}

func NewServer() *RPCSever {
	server := &RPCSever{}
	server.serviceMap = make(map[string]interface{})
	server.listener = nil
	return server
}

func (self *RPCSever) Serve(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	self.listener = &listener
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		// allocte a goroutine for each connection
		go func(c net.Conn) {
			var lasRequestSerialNumber uint64 = 0
			var lastReplyPtr interface{} = nil
			var err error
		ConnectionHandle:
			for {
				// make sure the same RPC request will only be excuted once
				lasRequestSerialNumber, lastReplyPtr, err = self.HandleRPCRequest(c,
					lasRequestSerialNumber, lastReplyPtr)
				if reflect.TypeOf(err) == reflect.TypeOf(NetworkError{}) {
					// if a network problem happened, give up the connection
					c.Close()
					break ConnectionHandle
				}
			}
			c.Close()
		}(conn)
	}
}

// rcvr is pointer of a instance of RPC type, should not be nil
func (self *RPCSever) Register(rcvr interface{}) error {
	if rcvr == nil {
		return fmt.Errorf("sevice instance should not be nil!")
	}

	t := reflect.TypeOf(rcvr)
	sname := t.Elem().Name()

	if _, in := self.serviceMap[sname]; in {
		return fmt.Errorf("sevice name is registered!\n")
	} else {
		for i := 0; i < t.NumMethod(); i++ {
			method := t.Method(i)
			// the reciever(pointer) will be regarded as a input parameter
			if method.Type.NumIn() != 3 {
				return fmt.Errorf("method parameters number must be 2!")
			}
			if method.Type.NumOut() != 1 {
				return fmt.Errorf("method output number must be 1!")
			}
		}
	}
	self.serviceMap[sname] = rcvr

	return nil
}

// return the request serial number and pointer to reply
func (self *RPCSever) HandleRPCRequest(conn net.Conn, lastRequestSerial uint64,
	lastReplyPtr interface{}) (requestSerialNumber uint64, replyPtr interface{}, err error) {
	requestSerialNumber, serviceMethod, parameters, _, err := RecieveRPCPacket(conn)
	if err != nil {
		// if recieve a bad RPC packet, must return something to notice the other side
		if reflect.TypeOf(err) == reflect.TypeOf(BadPacketError{}) {
			SendRPCPacket(conn, 0, "", nil, nil)
		}
		return 0, nil, err
	}

	if requestSerialNumber != lastRequestSerial {
		// handle the RPC request
		strs := strings.Split(serviceMethod, ".")
		service := strs[0]
		method := strs[1]

		serviceInstance, registered := self.serviceMap[service]
		if !registered {
			return 0, nil, fmt.Errorf("service %v has not been registered!", service)
		}

		// use reflection to call the method
		v := reflect.ValueOf(serviceInstance)
		m := v.MethodByName(method)
		if !m.IsValid() {
			return 0, nil, fmt.Errorf("%v is not implemented!", serviceMethod)
		} else {
			// this Elem() method can also recieved by a reclect.Type value...
			// return a element's type the reciever(as a pointer) point to
			// replyPtr is a pointer, reflect.New() returns a pointer
			replyPtr = reflect.New(m.Type().In(1).Elem()).Interface()

			// the first arguments of the method might be a pointer
			// in the situation, pass a pointer which point to the parameters as arguments
			var args reflect.Value
			if m.Type().In(0).Kind() == reflect.Ptr {
				args = reflect.New(reflect.TypeOf(parameters))
				args.Elem().Set(reflect.ValueOf(parameters))
			} else {
				args = reflect.ValueOf(parameters)
			}

			// call the method
			in := []reflect.Value{args, reflect.ValueOf(replyPtr)}
			m.Call(in)
		}

		reply := reflect.ValueOf(replyPtr).Elem().Interface()
		err = SendRPCPacket(conn, requestSerialNumber, serviceMethod, nil, reply)
		if err != nil {
			return 0, nil, err
		}
	} else {
		// if recieve a same RPC request with the last one
		// resend last reply directly without excuting
		err = SendRPCPacket(conn, requestSerialNumber, serviceMethod,
			nil, reflect.ValueOf(lastReplyPtr).Elem().Interface())
		if err != nil {
			return 0, nil, err
		}
	}

	return requestSerialNumber, replyPtr, nil
}
