package readSession

//this maybe should be in its own package

import "net"
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


func (s *readSession) run(par *ReadSessions) {
	numStarted := 0

	//TODO should send this but causing problems
	//s.sender.Send(tn.ComposeFirstData(s.useraddr))
Main:
	for {
	Inner:
		for numStarted < tn.WINDOW_SIZE {
			bnum, ok := s.findBlockNotStarted()
			if !ok {
				//maybe have none started, but not all finished
				if s.checkFinished() {
					fmt.Println("FINISHED SENDING")
					break Main
				}
				break Inner //important. without can lead to infinite loop
			} else {
				s.sendBlock(bnum)
				s.blocks[bnum] = 1
				numStarted++
			}
		}
		select {
		//TODO add a shutdown chan case for fast shutdown
		case block := <-s.newblock:
			fmt.Printf("newb\n")
			if s.blocks[block] == 1 {
				//fmt.Println("RECEIVED ACK FOR BLOCK:", block)
				s.blocks[block] = 2
				numStarted--
			} else {
				fmt.Println("received ack for block not requested. ignoring")
			}
		case timeoutblock := <-s.timeout:
			if s.blocks[timeoutblock] == 1 {
				fmt.Printf("timed out block:%v", timeoutblock)
				numStarted--
				s.blocks[timeoutblock] = 0 //set back to zero so loop can select again
			} else if s.blocks[timeoutblock] == 2 {
				//we must have already received this block and already decremented timeoutblock
			} else {
				panic(errors.New("timed out on a block not started. should not happen"))
			}
		case <-par.shutdown:
			fmt.Println("shutting down read session")
			par.shutdownok<-1
			break Main
		}


	}
}
func (s *readSession) findBlockNotStarted() (uint16, bool) {
	for i := 0; i < len(s.blocks); i++ {
		if s.blocks[i] == 0 {
			return uint16(i), true
		}
	}
	return 0, false
}
func (s *readSession) checkFinished() bool {
	for i := uint16(0); i < uint16(len(s.blocks)); i++ {
		if s.blocks[i] != 2 {
			return false
		}
	}
	return true
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
	s.sender.Send(tn.ComposeData(b, mm, s.useraddr))
}

func (s *readSession) finished() bool {
	for i := 0; i < len(s.blocks); i++ {
		if s.blocks[i] != 2 {
			return false
		}
	}
	return true
}
