package writeSession

//this maybe should be in its own package

import "net"
import "fmt"
import "errors"
import "sync"
import tn "reed/tftp_project/tftp_s/tftpnet"
import fi "reed/tftp_project/tftp_s/file"





//no two writeSessions will have same filename
//can use filename to identify writeSessions
type WriteSessions struct {
	wr   []writeSession
	lock sync.Mutex
}

func (b WriteSessions) String() string {
	return fmt.Sprintf("%v", b.wr)
}


//takes in a datapacket and adds it to the correct write session
//returns the finished file if that was the last packet we needed
//returns nil file if wasnt last
func (s *WriteSessions) HandleDataPacket(dpaddr *tn.Dpaddr) (*fi.File, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i := 0; i < len(s.wr); i++ {
		element := &s.wr[i]
		if tn.Eq(element.useraddr, dpaddr.Remoteaddr) {
			file, err := element.addBlock(dpaddr.Dp)

			return file, err
		}
	}
	return nil, errors.New("no addr")
}

func (s *WriteSessions) StartNewWriteSession(addr *net.UDPAddr, name string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	w := writeSession{useraddr: addr, filename: name, blocks: make([]bool, 0), lastBlock: nil, workdata: make([]byte, 0)}

	//todo make sure something
	for i := 0; i < len(s.wr); i++ {
		element := &s.wr[i]
		if tn.Eq(element.useraddr, w.useraddr) {
			return errors.New("user is already writing something")
		}
	}

	s.wr = append(s.wr, w)
	return nil
}
func (s *WriteSessions) CloseWriteSession(addr *net.UDPAddr) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	fmt.Println("Closing write session!")
	for i := 0; i < len(s.wr); i++ {
		element := &s.wr[i]
		if tn.Eq(element.useraddr, addr) {
			copy(s.wr[i:], s.wr[i+1:])
			s.wr = s.wr[:len(s.wr)-1]
			return nil
		}
	}
	return errors.New("cannot delete. write session not found")
}
