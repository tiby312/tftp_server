package writeSession

//this maybe should be in its own package

import "net"
import "fmt"
import "errors"
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


//for debuging
func (b writeSession) String() string {
	return fmt.Sprintf("%v , %v", b.blocks, b.lastBlock)
}


type DupBlockErr struct {
	Bnum uint16
}

func (d *DupBlockErr) Error() string {
	return "duplicate block"
}

//returns nil,nil if added block but wasnt last block to finish
//returns file,nil if added block was last block to finish
//returns nil,err otherwise
func (w *writeSession) addBlock(datap tn.Datapacket) (*fi.File, error) {

	//blocknum but counting from zero
	bnz := int(datap.Blocknum)

	//if we have encountered a blockindex larger than what we can fit already
	//make sure we grow to fit it
	if bnz >= len(w.blocks) {

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
		return nil, &DupBlockErr{Bnum: datap.Blocknum}
	}

	w.blocks[bnz] = true

	nn := bnz * tn.BLOCK_SIZE
	copy(w.workdata[nn:int(nn)+len(datap.Data)], datap.Data)

	if len(datap.Data) == 0 {
		//this could be allowed
		//panic(errors.New("length of data is zero. did user try to uploade an empty file?"))
	}
	if len(datap.Data) < tn.BLOCK_SIZE {
		if w.lastBlock != nil {
			panic(errors.New("received two blocks less than fixed block length"))
		}
		lb := lastBlockD{index: uint16(bnz), size: uint16(len(datap.Data))}
		w.lastBlock = &lb

		fmt.Printf("found last block %v\n", *(w.lastBlock))
	}

	if w.finished() {
		numblocks := int(w.lastBlock.index) + 1
		sizeofrest := (numblocks - 1) * tn.BLOCK_SIZE
		sizeoflast := int(w.lastBlock.size)
		totalsize := sizeofrest + sizeoflast
		//var ss int64 = (int64(w.lastBlock.index)+1)*tn.BLOCK_SIZE - (tn.BLOCK_SIZE - int64(w.lastBlock.size))
		file := fi.File{Data: make([]byte, totalsize), Name: w.filename}
		copy(file.Data, w.workdata[0:totalsize])
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
