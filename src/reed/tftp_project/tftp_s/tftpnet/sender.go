package tftpnet

import "net"

import "encoding/binary"
import "fmt"

const (
	OPCODE_RRQ   = 1
	OPCODE_WRQ   = 2
	OPCODE_DATA  = 3
	OPCODE_ACK   = 4
	OPCODE_ERROR = 5
	BLOCK_SIZE   = 512
)

const (
	WINDOW_SIZE   = 10 //how many blocks do you want to sender before waiting for acks
	BLOCK_TIMEOUT = 2  //in seconds. when to try resend a block to a client
	DALLY_TIME    = 10 //in seconds. how long to stick around after receiving a file
)

type error_code int

const (
	//ERR_UNDEFINED=0
	ERR_FILE_NOT_FOUND = 1
	//ERR_ACCESS_VIOLATION=2
	//ERR_DISKFULL=3
	//ERR_ILLEGAL_TFTP_OP=4
	//ERR_UNKNOWN_TRANSFER_ID=5
	ERR_FILE_EXISTS = 6
	//ERR_NOSUCH_USER=7
)

//sender object has lowish level methods like sendack or parse tftpp's
type Sender struct {
	conn      *net.UDPConn
	localaddr *net.UDPAddr
	buf       []byte      //once buffer is used to read in all udp packets
	outbox    chan *Tftpp //private to disallow anyone but sender reading
}

//when receiving packages
//every tftp packet is first parsed as one of these.
//To send a packet, must put into this form first
type Tftpp struct {
	Opcode     uint16
	Payload    []byte //with opcode
	Remoteaddr *net.UDPAddr
}

func (s *Sender) GetPort() int {
	return s.localaddr.Port
}

//private constructor
func createSender(con *net.UDPConn, addr *net.UDPAddr) Sender {
	return Sender{
		outbox:    make(chan *Tftpp, 10),
		conn:      con,
		localaddr: addr,
		buf:       make([]byte, 1024)} //todo check if this buffer is right size
}
func CreateSenderRandPort() Sender {
	con, add := FindPort()
	return createSender(con, add)
}
func CreateSender(port int) (Sender, bool) {
	addr := net.UDPAddr{Port: port, IP: net.ParseIP("localhost")}
	conn2, err := net.ListenUDP("udp", &addr)

	if err != nil {
		return Sender{}, false
	} else {
		return createSender(conn2, &addr), true
	}
}

//run this as a go routine.
func (s *Sender) Run() {
	for {
		o := <-s.outbox

		_, err := s.conn.WriteToUDP(o.Payload, o.Remoteaddr)
		if err != nil {
			fmt.Println(err)
			fmt.Printf("payload size:%v\n", len(o.Payload))
			//panic(err)
		}
		fmt.Println("success sent")
	}
}

//push a message to send asynchronously onto outbox
func (s *Sender) Send(t *Tftpp) {
	s.outbox <- t
}

//TODO should be somewhere else
func (s *Sender) Get_next_tftpp() (*Tftpp, error) {
	rlen, addr, err := s.conn.ReadFromUDP(s.buf)
	if err != nil {
		return nil, err
	}
	opcode := binary.BigEndian.Uint16(s.buf[0:2])
	bla := make([]byte, rlen)
	copy(bla, s.buf[0:rlen])

	pp := Tftpp{Opcode: opcode, Payload: bla, Remoteaddr: addr}
	return &pp, err
}

//starts at port 1000 and incrementally looks for an open port
func FindPort() (conn *net.UDPConn, addr *net.UDPAddr) {
	c := 1000
	for {
		addr := net.UDPAddr{Port: c, IP: net.ParseIP("localhost")}
		conn2, err := net.ListenUDP("udp", &addr)

		if err != nil {
			fmt.Println(err)
		} else {
			return conn2, &addr
		}
		c++
	}
}
