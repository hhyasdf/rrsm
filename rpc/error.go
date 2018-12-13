package rpc

type NetworkError struct{}

type BadPacketError struct{}

func (self NetworkError) Error() string {
	return "network problems happened!"
}

func (self BadPacketError) Error() string {
	return "recieve a broken RPC packet!"
}
