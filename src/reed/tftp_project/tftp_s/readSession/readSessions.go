package readSession

//this maybe should be in its own package

import "net"
import "sync"
import "errors"
import "fmt"
import tn "reed/tftp_project/tftp_s/tftpnet"
import fi "reed/tftp_project/tftp_s/file"


type ReadSessions struct {
	rr   []*readSession
	lock sync.Mutex
	shutdown chan int
	shutdownok chan int	
}

func Create() ReadSessions{
	a:=ReadSessions{rr:make([]*readSession,0),shutdown:make(chan int),shutdownok:make(chan int)}
	return a
}



//blocks until shutdown
func (s *ReadSessions) Stop() {
	//notify active readsessions that they should shutdown
	for i:=0;i<len(s.rr);i++ {
		s.shutdown<-1
	}
	for i:=0;i<len(s.rr);i++ {
		<-s.shutdownok
	}	
	fmt.Println("success shutdown read sessions")	
}



func (s *ReadSessions) HandleAck(num uint16, addr *net.UDPAddr) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	reads := s.findReadSession(addr)
	if reads == nil {
		fmt.Println(errors.New("got an ack we don't need"))
	}
	if num >= uint16(len(reads.blocks)) {
		panic(errors.New("block index out of bounds"))
	}

	fmt.Println("pushing ack onto chan")
	reads.newblock <- num
	fmt.Println("pushed ack onto chan")
	return nil
}
func (s *ReadSessions) findReadSession(addr *net.UDPAddr) *readSession {
	for i := 0; i < len(s.rr); i++ {
		if tn.Eq(s.rr[i].useraddr, addr) {
			return s.rr[i]
		}
	}
	return nil
}

//TODO multiple people should allowed to read same file

func (s *ReadSessions) StartNewReadSessionAndRun(addr *net.UDPAddr, filename string, filesys *fi.Files, sender *tn.Sender) {
	s.lock.Lock()
	defer s.lock.Unlock()

	numb, err := filesys.GetNumBlocks(filename)
	if err != nil {
		panic(err)
	}
	r := &readSession{
		sender:   sender,
		useraddr: addr,
		filesys:  filesys,
		filename: filename,
		blocks:   make([]uint8, numb),
		newblock: make(chan uint16),
		timeout:  make(chan uint16)}

	s.rr = append(s.rr, r)
	go r.run(s)
}
