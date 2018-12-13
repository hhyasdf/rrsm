package rpc

/*
 * implement a simple rpc
 * need to make sure the RPC connection is available
 */

// port is a port of localhost
// can register multi rpc types once
func RPCRegister(port string, rpcTypes ...interface{}) error {
	server := NewServer()
	for _, obj := range rpcTypes {
		server.Register(obj)
	}

	go server.Serve(port)
	return nil
}

// serverAddress is a form of ip:port or hostname:port
func RPCConnect(serverAddress string) (*RPCClient, error) {
	client := NewClient()
	err := client.Connect(serverAddress)
	return client, err
}

func RPCCall(client *RPCClient, serviceMethod string, obj interface{}, reply interface{}) error {
	// what will the Client.Call do if a sever is down and then raised up again?
	return client.Call(serviceMethod, obj, reply)
}
