package tftpnet

import "net"
import "fmt"

//use custom type to remind to
//start counting from one and not zero
type Blockindex uint16

type Datapacket struct {
	Data     []byte
	Blocknum Blockindex
}

type Dpaddr struct {
	Dp         Datapacket
	Remoteaddr *net.UDPAddr
}

func (s *Dpaddr) String() string {
	return fmt.Sprintf("%v", s.Dp.Blocknum) //string(int(s.Dp.Blocknum))
}
