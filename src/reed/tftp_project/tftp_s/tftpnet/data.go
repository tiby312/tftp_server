package tftpnet

import "net"
import "fmt"



//check if two addresses are the same. not sure if this is the correct way to show equality
func Eq(a *net.UDPAddr, b *net.UDPAddr) bool {
	return a.String() == b.String()
}



/*
tftp packets that contain data are parsed into these data structures
*/
type Datapacket struct {
	Data     []byte
	Blocknum uint16
}

type Dpaddr struct {
	Dp         Datapacket
	Remoteaddr *net.UDPAddr
}

func (s *Dpaddr) String() string {
	return fmt.Sprintf("%v", s.Dp.Blocknum) //string(int(s.Dp.Blocknum))
}
