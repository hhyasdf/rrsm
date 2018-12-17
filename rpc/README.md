## Overview ##
This is a simple implement of RPC framework. It has a serie of functions which can be ragarded as a small subset of the official package named *net/rpc* in golang.

## Features ##
- Simple usage
- Based on TCP protocol.
- Maintain a long TCP connection for upper caller, will auto-rebuild connection and return a NetworkError when network problems happen.
- Thread-safe operation

## Usages ##

Just like the normal usages of *net/rpc* :

```go
import (
	"github.com/hhyasdf/rrsm/rpc"
)

// generate a RPC sever
server := rpc.NewServer()

// register a RPC service, just like net/rpc, cannot register the same type (name) 
// of service twice. And the requirments of method is almost the same as net/rpc, 
// the Service struct need to be able to be encode/decode by encoding/gob, and the
// field name of Service struct must be exported (capitalized)
err := server.Register(&Service{})

// the Serve() function accept a string which is port number of the localhost the
// server listen to, a port cannot be listened by two server
server.Serve(portNumber)

// generate a RPC client
client := rpc.NewClient()

// connct to a RPC server, address is a string type, need to including port number
// just like "ip:port"
client.Connect(address)

// client call a method. the method name must be capitalized, and accepts two 
// arguments. The first argument can both be a pointer or not pointer, if it is 
// (or is a pointer of) a struct, the struct also need to be able to be encode/decode 
// by encoding/gob. And the second argument must be a pointer, which points to a empty
// built-in type value or struct (need to be able to be encode/decode by encoding/gob)
// used to fill the return value, and differs for the implement of net/rpc, the Call()
// method return a error of network or system if exits, rather than the RPC error
err := c.Call("Service.MethodName", &args, &reply)  // it's a Thread-safe operation
```

More simple usages is available:

```go
// register multi rpc types once and create a server listen to a port
func RPCRegister(port string, rpcTypes ...interface{}) error

// return a RPC client and conect to a server
func RPCConnect(serverAddress string) (*RPCClient, error)

// make a RPC call, return a error of network or system
func RPCCall(client *RPCClient, serviceMethod string, args interface{}, reply interface{})
```

Referncing to the *test* directory, under which is two examples of using the rpc framework. A more complicated applyment see [raft rpc](https://github.com/hhyasdf/rrsm/blob/master/rrpc.go) and [send heartbeat](https://github.com/hhyasdf/rrsm/blob/master/node.go#L160)

### Reference ###
- [Golang net/rpc](https://golang.org/pkg/net/rpc/)
