package rpc

/*
 * use gob as the serialization tool, because json will transfor int to float after serialization
 */

import (
	"bytes"
	"encoding/gob"
	"hash/crc32"
	"io"
	"reflect"
)

type RPCPacket struct {
	RequestSerialNumber uint64
	ServiceMethod       string
	Parameters          interface{}
	Reply               interface{}
	Checksum            uint32
}

// if there is an error while reading from socket, it's believed a network error,
// return a NetworkError instance, if a bad checksum appears, return a BadPacketError
func RecieveRPCPacket(conn io.Reader) (requestSerialNum uint64, serverMethod string,
	args interface{}, reply interface{}, err error) {
	packet := RPCPacket{}
	decoder := gob.NewDecoder(conn)

	err = decoder.Decode(&packet)
	if err != nil {
		return 0, "", nil, nil, NetworkError{}
	}

	// check the checksum
	originChecksum := packet.Checksum
	packet.Checksum = 0

	buffer := new(bytes.Buffer)
	bin := gob.NewEncoder(buffer)

	err = bin.Encode(packet)
	if err != nil {
		return 0, "", nil, nil, err
	}
	if crc32.ChecksumIEEE(buffer.Bytes()) != originChecksum {
		return 0, "", nil, nil, BadPacketError{}
	}

	return packet.RequestSerialNumber, packet.ServiceMethod, packet.Parameters, packet.Reply, nil
}

// send a PRC packet and compute a checksum, if an error happened while writing to socket
// return a NetworkError instance.
func SendRPCPacket(conn io.Writer, requestSerialNumber uint64, method string, args interface{},
	reply interface{}) error {
	buffer := new(bytes.Buffer)

	GobRegister(args, reply)
	bin := gob.NewEncoder(buffer)

	// compute the checksum
	newPacket := RPCPacket{
		RequestSerialNumber: requestSerialNumber,
		ServiceMethod:       method,
		Parameters:          args,
		Reply:               reply,
		Checksum:            0,
	}

	err := bin.Encode(newPacket)
	if err != nil {
		return err
	}

	newPacket.Checksum = crc32.ChecksumIEEE(buffer.Bytes())

	// write packet to socket
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(newPacket)
	if err != nil {
		return NetworkError{}
	}

	return nil
}

// gob need to register... reference to the gob documents
func GobRegister(rcvr ...interface{}) {
	for _, item := range rcvr {
		if item != nil {
			v := reflect.ValueOf(item)
			gob.Register(reflect.Indirect(v).Interface())
		}
	}
}
