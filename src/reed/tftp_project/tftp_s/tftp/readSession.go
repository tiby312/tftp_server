package tftp

import "net"
import "sync"
import "time"
import "errors"
import "fmt"
import tn "reed/tftp_project/tftp_s/tftpnet"
import fi "reed/tftp_project/tftp_s/file"

//one read session that is in the middle of writing
type readSession struct {
	useraddr *net.UDPAddr

	//0 if not started
	//1 if started
	//2 if finished
	blocks []uint8 //blocks that have been acked

	//file     *file
	filesys  *fi.Files
	filename string
	sender   *tn.Sender
	newblock chan uint16
	timeout  chan uint16
}

type readSessions struct {
	rr   []*readSession
	lock sync.Mutex
}

func (s *readSession) run() {
	numStarted := 0
Main:
	for {
		fmt.Println("loop")
		for numStarted < tn.WINDOW_SIZE {
			bnum, fin := s.findBlockNotStarted()
			if fin {
				fmt.Println("FINISHED SENDING")
				break Main
			}
			s.sendBlock(bnum)
			s.blocks[bnum] = 1
			numStarted++
		}
		fmt.Println("waiting for acks")
		select {
		case block := <-s.newblock:
			if s.blocks[block] == 1 {
				fmt.Println("RECEIVED ACK FOR BLOCK:", block)
				s.blocks[block] = 2
				numStarted--
			} else {
				fmt.Println("received ack for block not requested. ignoring")
			}
		case timeoutblock := <-s.timeout:
			fmt.Println("TIMED OUT")
			if s.blocks[timeoutblock] == 1 {
				numStarted--
				s.blocks[timeoutblock] = 0 //set back to zero so loop can select again
			} else if s.blocks[timeoutblock] == 2 {
				//we must have already received this block and already decremented timeoutblock
			} else {
				panic(errors.New("timed out on a block not started. should not happen"))
			}
		}

	}
}
func (s *readSession) findBlockNotStarted() (uint16, bool) {
	for i := uint16(0); i < uint16(len(s.blocks)); i++ {
		if s.blocks[i] == 0 {
			return i, false
		}
	}
	return 0, true
}
func (s *readSession) sendBlock(b uint16) {

	fu := func() {
		timer := time.NewTimer(time.Second * tn.BLOCK_TIMEOUT)
		<-timer.C
		//todo add a select and wait for shutdown chan as well
		//to instantaniously shutodwn server
		s.timeout <- b
	}
	go fu()

	mm, err := s.filesys.GetBlock(s.filename, b)
	if err != nil {
		//todo handle
		panic(err)
	}
	//fmt.Println("sending block ii:%v  (%v,%v)", b, bs, be)
	bb := tn.Blockindex(b + 1)
	s.sender.Send(tn.ComposeData(bb, mm, s.useraddr))
}
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
func (s *readSession) finished() bool {
	for i := 0; i < len(s.blocks); i++ {
		if s.blocks[i] != 2 {
			return false
		}
	}
	return true
}

func (s *readSessions) handleAck(bind tn.Blockindex, addr *net.UDPAddr) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	reads := s.findReadSession(addr)
	if reads == nil {
		fmt.Println(errors.New("got an ack we don't need"))
	}
	num := uint16(bind) - 1
	if num >= uint16(len(reads.blocks)) {
		panic(errors.New("block index out of bounds"))
	}

	fmt.Println("pushing ack onto chan")
	reads.newblock <- num
	return nil
}
func (s *readSessions) findReadSession(addr *net.UDPAddr) *readSession {
	for i := 0; i < len(s.rr); i++ {
		if eq(s.rr[i].useraddr, addr) {
			return s.rr[i]
		}
	}
	return nil
}

//multiple people are allowed to read same file
func (s *readSessions) StartNewReadSessionAndRun(addr *net.UDPAddr, filename string, filesys *fi.Files, sender *tn.Sender) {
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
	fmt.Println("about to run reas sess")
	go r.run()
}
