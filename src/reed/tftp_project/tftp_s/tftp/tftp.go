package tftp

import "fmt"
import tn "reed/tftp_project/tftp_s/tftpnet"
import fi "reed/tftp_project/tftp_s/file"
import "time"

type Server struct {
	wsessions writeSessions
	rsessions readSessions
	files     fi.Files
	sender    tn.Sender
	Finished  chan int
}

func CreateServer(port int) (*Server, bool) {
	sender, ok := tn.CreateSender(port)
	if !ok {
		return nil, false
	}
	s := Server{
		sender:   sender,
		Finished: make(chan int),
		files:    fi.CreateFileSys()}
	return &s, true
}
func CreateServerRandPort() *Server {
	s := Server{
		sender:   tn.CreateSenderRandPort(),
		Finished: make(chan int),
		files:    fi.CreateFileSys()}
	return &s
}

func (s *Server) Run() {
	go s.sender.Run()
	fmt.Printf("Listening on port:%v\n", s.sender.GetPort())
	for {
		tftpp, err := s.sender.Get_next_tftpp()
		if err != nil {
			fmt.Printf("failed to parse")
			continue
		}
		go s.handlePacket(tftpp)
	}
}

func (s *Server) handlePacket(tftpp *tn.Tftpp) {
	switch tftpp.Opcode {

	case tn.OPCODE_RRQ:
		filename, mode, err := tn.ParseAsWRQorRRQ(tftpp)
		fmt.Printf("in:RRQ:%v mode:%v\n", filename, mode)
		mode = mode
		if err != nil {
			panic(err)
		}
		if !s.files.Exists(filename) {
			fmt.Println("file does not exist")
			s.sender.Send(tn.ComposeError(tn.ERR_FILE_NOT_FOUND, "could not find file"+filename, tftpp.Remoteaddr))
			return
		}
		fmt.Println("starting new read session")
		s.rsessions.StartNewReadSessionAndRun(tftpp.Remoteaddr, filename, &s.files, &s.sender)
	case tn.OPCODE_ACK:
		blocknum := tn.ParseAck(tftpp)
		fmt.Printf("in:ACK %v\n", blocknum)
		s.rsessions.handleAck(blocknum, tftpp.Remoteaddr)
	case tn.OPCODE_WRQ: //wrq
		filename, mode, err := tn.ParseAsWRQorRRQ(tftpp)
		//todo make sure mode is octet
		fmt.Printf("in:WRQ:%v mode:%v\n", filename, mode)
		if err != nil {
			panic(err)
		}
		if s.files.Exists(filename) { //make sure not downloading also
			//fmt.Printf("file already in filesystem. ignoring")
			s.sender.Send(tn.ComposeError(tn.ERR_FILE_EXISTS, "file exists:"+filename, tftpp.Remoteaddr))
			return
		}
		err = s.wsessions.StartNewWriteSession(tftpp.Remoteaddr, filename)
		if err != nil { //TODO make sure filename not also being read
			fmt.Printf("already in middle of being written by someone. ignoring")
			return
		}
		s.sender.Send(tn.ComposeDataAck(tftpp.Remoteaddr, 0))
	case tn.OPCODE_DATA: //data
		dpaddr := tn.ParseAsData(tftpp)
		fmt.Printf("in:data:%v\n", dpaddr.String())
		file, err := s.wsessions.HandleDataPacket(dpaddr)
		if err != nil {
			switch t := err.(type) {
			case *DupBlockErr:
				s.sender.Send(tn.ComposeDataAck(tftpp.Remoteaddr, t.bnum))
			}
			fmt.Println(err)
			return
		}
		s.sender.Send(tn.ComposeDataAck(tftpp.Remoteaddr, dpaddr.Dp.Blocknum))
		if file != nil {
			err := s.files.Add(file)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}

			//close wsession after dally time
			//this timer should really be set every time
			//we receive a tftp packet from client.
			f := func() {
				timer := time.NewTimer(time.Second * tn.DALLY_TIME)
				<-timer.C
				//todo add switch on shutdown chan so we can shutdown server
				//elegantly
				s.wsessions.CloseWriteSession(tftpp.Remoteaddr)
			}
			go f()
		}
	}
}
