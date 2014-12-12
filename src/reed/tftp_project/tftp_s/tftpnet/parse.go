package tftpnet

/*
Has low level code for parsing blocks.

Note: in this package, blocknums start at 1.
Outside of this package, blocknums start at 0.
So to the rest of the code, its as if they started from zero.
But when they are send through the protocol, they start at 1.
*/

import "encoding/binary"
import "strings"
import "errors"
import "net"

func ParseAsData(tftpp *Tftpp) *Dpaddr {
	num := uint16(binary.BigEndian.Uint16(tftpp.Payload[2:4]))
	num -= 1
	dp := Datapacket{Blocknum: num, Data: tftpp.Payload[4:len(tftpp.Payload)]}
	dpaddr := Dpaddr{Dp: dp, Remoteaddr: tftpp.Remoteaddr}
	return &dpaddr
}
func ComposeData(bind uint16, data []byte, addr *net.UDPAddr) *Tftpp {
	//a := tftpp{}
	bb := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(bb, OPCODE_DATA)
	binary.BigEndian.PutUint16(bb[2:4], bind+1)
	copy(bb[4:], data)
	a := Tftpp{Opcode: OPCODE_ACK, Payload: bb, Remoteaddr: addr}
	return &a
}
func ParseAsWRQorRRQ(tftpp *Tftpp) (filename string, mode string, err error) {
	i := 2
	fok := false
	mok := false
	for ; i < len(tftpp.Payload); i++ {
		if tftpp.Payload[i] == 0 {
			filename = strings.TrimSpace(string(tftpp.Payload[2:i]))
			fok = true
			break
		}
	}
	i++
	starti := i
	for ; i < len(tftpp.Payload); i++ {
		if tftpp.Payload[i] == 0 {
			mode = strings.TrimSpace(string(tftpp.Payload[starti:i]))
			mok = true
			break
		}
	}
	if fok && mok {
		err = nil
		return
	} else {
		err = errors.New("could not parse")
		return
	}
}

func ComposeError(code error_code, str string, addr *net.UDPAddr) *Tftpp {
	bb := make([]byte, 4+len(str)+1)
	binary.BigEndian.PutUint16(bb, OPCODE_ERROR)
	binary.BigEndian.PutUint16(bb[2:4], uint16(code))
	copy(bb[4:len(str)], str)
	bb[4+len(str)] = 0
	vv := Tftpp{Opcode: OPCODE_ERROR, Payload: bb, Remoteaddr: addr}
	return &vv
}
func ParseAck(tftpp *Tftpp) uint16 {
	num := uint16(binary.BigEndian.Uint16(tftpp.Payload[2:4]))
	num -= 1 //sinceoffset
	return num
}
func ComposeDataAck(addr *net.UDPAddr, blocknum uint16) *Tftpp {
	bb := make([]byte, 4)
	//PutUint16([]byte, uint16)
	binary.BigEndian.PutUint16(bb, OPCODE_ACK)
	binary.BigEndian.PutUint16(bb[2:4], blocknum+1)
	vv := Tftpp{Opcode: OPCODE_ACK, Payload: bb, Remoteaddr: addr}
	return &vv
}
