package file

import "errors"
import "sync"
import "fmt"
import "bytes"
import tn "reed/tftp_project/tftp_s/tftpnet"

type File struct {
	Name string
	Data []byte
}

type Files struct {
	files []File
	lock  sync.Mutex
}

func CreateFileSys() Files {
	f := Files{files: make([]File, 0)}
	return f
}

func (file *File) numBlocks() uint16 {
	vv := len(file.Data) / tn.BLOCK_SIZE
	if vv < len(file.Data) {
		vv += 1
	}
	return uint16(vv)
}

func (f *File) getBlock(b uint16) ([]byte, error) {
	if b >= f.numBlocks() {
		return nil, errors.New("block index not in file")
	}
	bs := b * tn.BLOCK_SIZE
	be := len(f.Data)
	return f.Data[bs:be], nil
}

func (f *Files) String() string {
	var buffer bytes.Buffer
	for i := 0; i < len(f.files); i++ {
		buffer.WriteString(fmt.Sprintf("%s,", f.files[i].Name))
	}
	return buffer.String()
}

func (f *Files) Add(file *File) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	if f.exists(file.Name) {
		return errors.New("file exists already")
	}
	f.files = append(f.files, *file)
	fmt.Printf("added file:%v\n", file.Name)
	fmt.Printf("files:%s\n", f.String())
	return nil

}
func (f *Files) GetNumBlocks(name string) (uint16, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	file := f.get(name)
	if file == nil {
		return 0, errors.New("file not found")
	}
	return file.numBlocks(), nil

}
func (f *Files) GetBlock(name string, block uint16) ([]byte, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	file := f.get(name)
	if file == nil {
		return nil, errors.New("can't find file")
	}
	return file.getBlock(block)
}
func (f *Files) Exists(name string) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	return f.exists(name)
}

func (f *Files) get(name string) *File {
	for i := 0; i < len(f.files); i++ {
		if f.files[i].Name == name {
			return &(f.files[i])
		}
	}
	return nil
}
func (f *Files) exists(name string) bool {

	for _, element := range f.files {
		if element.Name == name {
			return true
		}
	}
	return false
}
