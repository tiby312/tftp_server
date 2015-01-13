package tftp

import "fmt"
import rs "reed/tftp_project/tftp_s/readSession"
import ws "reed/tftp_project/tftp_s/writeSession"
import tn "reed/tftp_project/tftp_s/tftpnet"
import fi "reed/tftp_project/tftp_s/file"
import "time"

type Server struct {
	wsessions ws.WriteSessions
	rsessions rs.ReadSessions
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
		rsessions: rs.Create(),
		sender:   sender,
		Finished: make(chan int),
		files:    fi.CreateFileSys(),
	}
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
	Loop:for {
		select{
		case <-s.Finished:
			fmt.Println("finished!");
			break Loop;
		default:
			tftpp, err := s.sender.Get_next_tftpp()
			if err != nil {
				fmt.Printf("failed to parse")
				continue
			}
			go s.handlePacket(tftpp)
		}
		
	}
}
func (s *Server) Stop(){
	s.rsessions.Stop();
	s.sender.Stop();
	s.Finished<-1
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
		s.rsessions.HandleAck(blocknum, tftpp.Remoteaddr)
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
		s.sender.Send(tn.ComposeFirstDataAck(tftpp.Remoteaddr))
	case tn.OPCODE_DATA: //data
		dpaddr := tn.ParseAsData(tftpp)
		fmt.Printf("in:data:%v\n", dpaddr.String())
		file, err := s.wsessions.HandleDataPacket(dpaddr)
		if err != nil {
			switch t := err.(type) {
			case *ws.DupBlockErr:
				s.sender.Send(tn.ComposeDataAck(tftpp.Remoteaddr, t.Bnum))
			}
			fmt.Println(err)
			return
		}
		//fmt.Printf("%s\n", s.wsessions.String())
		s.sender.Send(tn.ComposeDataAck(tftpp.Remoteaddr, dpaddr.Dp.Blocknum))
		if file != nil {
			err := s.files.Add(file)
			if err != nil {
				fmt.Println(err)
				panic(err)
			}

			//for debuging
			//s.files.WriteToDisk(file.Name)

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
