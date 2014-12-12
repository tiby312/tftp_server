package tftp

import "net"
import "fmt"
import "errors"
import "sync"
import tn "reed/tftp_project/tftp_s/tftpnet"
import fi "reed/tftp_project/tftp_s/file"

//describes the last block
type lastBlockD struct {
	index uint16
	size  uint16
}

//one write session that is in the middle of writing
type writeSession struct {
	useraddr  *net.UDPAddr
	blocks    []bool
	workdata  []byte
	filename  string
	lastBlock *lastBlockD //is null if we havent found the last block yet
}

//no two writeSessions will have same filename
//can use filename to identify writeSessions
type writeSessions struct {
	wr   []writeSession
	lock sync.Mutex
}

//for debuging
func (b writeSession) String() string {
	return fmt.Sprintf("%v , %v", b.blocks, b.lastBlock)
}

func (b writeSessions) String() string {
	return fmt.Sprintf("%v", b.wr)
}

func (s *writeSessions) StartNewWriteSession(addr *net.UDPAddr, name string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	w := writeSession{useraddr: addr, filename: name, blocks: make([]bool, 0), lastBlock: nil, workdata: make([]byte, 0)}

	//todo make sure something
	for i := 0; i < len(s.wr); i++ {
		element := &s.wr[i]
		if eq(element.useraddr, w.useraddr) {
			return errors.New("user is already writing something")
		}
	}

	s.wr = append(s.wr, w)
	return nil
}
func (s *writeSessions) CloseWriteSession(addr *net.UDPAddr) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	fmt.Println("Closing write session!")
	for i := 0; i < len(s.wr); i++ {
		element := &s.wr[i]
		if eq(element.useraddr, addr) {
			copy(s.wr[i:], s.wr[i+1:])
			s.wr = s.wr[:len(s.wr)-1]
			return nil
		}
	}
	return errors.New("cannot delete. write session not found")
}

//check if two addresses are the same. not sure if this is the correct way to show equality
func eq(a *net.UDPAddr, b *net.UDPAddr) bool {
	return a.String() == b.String()
}

//takes in a datapacket and adds it to the correct write session
//returns the finished file if that was the last packet we needed
//returns nil file if wasnt last
func (s *writeSessions) HandleDataPacket(dpaddr *tn.Dpaddr) (*fi.File, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for i := 0; i < len(s.wr); i++ {
		element := &s.wr[i]
		if eq(element.useraddr, dpaddr.Remoteaddr) {
			file, err := element.addBlock(dpaddr.Dp)

			// if file != nil {
			// 	copy(s.wr[i:], s.wr[i+1:])
			// 	s.wr = s.wr[:len(s.wr)-1]
			// }
			return file, err
		}
	}
	return nil, errors.New("no addr")
}

type DupBlockErr struct {
	bnum tn.Blockindex
}

func (d *DupBlockErr) Error() string {
	return "duplicate block"
}

//returns nil,nil if added block but wasnt last block to finish
//returns file,nil if added block was last block to finish
//returns nil,err otherwise
func (w *writeSession) addBlock(datap tn.Datapacket) (*fi.File, error) {

	//blocknum but counting from zero
	bnz := uint16(datap.Blocknum - 1)

	//if we have encountered a blockindex larger than what we can fit already
	//make sure we grow to fit it
	if bnz >= uint16(len(w.blocks)) {

		//grow block by factor of 2
		nbs := (bnz + 1) * 2
		newblocks := make([]bool, nbs)
		copy(newblocks, w.blocks)
		w.blocks = newblocks[0:nbs]

		//grow workdata by factor of 2
		var nds uint64 = uint64(bnz+1) * tn.BLOCK_SIZE * 2

		newd := make([]byte, nds)
		copy(newd, w.workdata)
		w.workdata = newd[0:nds]
	}

	if w.blocks[bnz] {
		//already have this block reset ack
		return nil, &DupBlockErr{bnum: datap.Blocknum}
	}

	w.blocks[bnz] = true

	nn := bnz * tn.BLOCK_SIZE
	copy(w.workdata[nn:int(nn)+len(datap.Data)], datap.Data)

	if len(datap.Data) == 0 {
		//this could be allowed
		panic(errors.New("length of data is zero. did user try to uploade an empty file?"))
	}
	if len(datap.Data) < tn.BLOCK_SIZE {
		if w.lastBlock != nil {
			panic(errors.New("received two blocks less than fixed block length"))
		}
		lb := lastBlockD{index: bnz, size: uint16(len(datap.Data))}
		w.lastBlock = &lb

		fmt.Printf("found last block %v\n", *(w.lastBlock))
	}

	if w.finished() {
		ss := (w.lastBlock.index+1)*tn.BLOCK_SIZE - (tn.BLOCK_SIZE - w.lastBlock.size)
		file := fi.File{Data: make([]byte, ss), Name: w.filename}
		copy(file.Data, w.workdata[0:ss])
		return &file, nil
	} else {
		return nil, nil
	}

}
func (w *writeSession) finished() bool {
	if w.lastBlock != nil {
		//start at one since first data block at index 1.
		for i := uint16(0); i <= w.lastBlock.index; i++ {
			if !w.blocks[i] {
				return false
			}
		}
		return true
	}
	return false
}
