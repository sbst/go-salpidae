package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
)

type blockError struct {
	BlockId int
	Err     error
}

func (e *blockError) Error() string {
	return fmt.Sprintf("Block %v error: %v", e.BlockId, e.Err.Error())
}

func (e *blockError) Unwrap() error { return e.Err }

func getNrOfBlocks(size int64, blockSize int) int {
	return int(math.Ceil(float64(size) / float64(blockSize)))
}

func getFileSize(fileName string) (int64, error) {
	f, e := os.Open(fileName)
	if e != nil {
		return 0, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return 0, e
	}
	return info.Size(), nil
}

func skipBlocks(file *os.File, blockSize int, nrOfBlocks int) error {
	_, e := file.Seek(int64(blockSize)*int64(nrOfBlocks), io.SeekStart)
	return e
}

func read(reader io.Reader, blockSize int, startBlockId int, nrBlocks int, result []string) error {
	hasher := sha256.New()
	block := make([]byte, blockSize)
	curBlockId := startBlockId
	for nrBlocks > 0 {
		n, e := io.ReadFull(reader, block)
		if n == 0 && e == io.EOF {
			break
		} else if e == io.ErrUnexpectedEOF {
			block = block[0:n]
			e = nil
		} else if e != nil {
			return &blockError{curBlockId, e}
		}
		_, e = hasher.Write(block)
		if e != nil {
			return &blockError{curBlockId, e}
		}
		result[curBlockId] = fmt.Sprintf("%x", hasher.Sum(nil))
		hasher.Reset()
		curBlockId += 1
		nrBlocks -= 1
	}
	return nil
}

type errorList struct {
	mutex  sync.Mutex
	errors []error
}

func (list *errorList) add(err error) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
	list.errors = append(list.errors, err)
}

func (list *errorList) get(i uint) error {
	list.mutex.Lock()
	defer list.mutex.Unlock()
	if i+1 <= uint(len(list.errors)) {
		return list.errors[i]
	}
	return nil
}

func (list *errorList) isEmpty() bool {
	list.mutex.Lock()
	defer list.mutex.Unlock()
	return len(list.errors) == 0
}

func readFileExec(fileName string, blockSize int, startBlockId int, nrBlocks int, errors *errorList, wait *sync.WaitGroup, result []string) {
	defer wait.Done()
	f, e := os.Open(fileName)
	if e != nil {
		errors.add(e)
		return
	}
	defer f.Close()

	e = skipBlocks(f, blockSize, startBlockId)
	if e != nil {
		errors.add(e)
		return
	}

	e = read(f, blockSize, startBlockId, nrBlocks, result)
	if e != nil {
		errors.add(e)
		return
	}
}

func readFile(fileName string, blockSize int, nrBlocksPerThread int) ([]string, error) {
	var wait = sync.WaitGroup{}
	fileSize, e := getFileSize(fileName)
	if e != nil {
		return make([]string, 0), e
	}
	var errors errorList
	nrBlocksTotal := getNrOfBlocks(fileSize, blockSize)
	hashes := make([]string, nrBlocksTotal)
	var processedBlocks int = 0
	for processedBlocks < nrBlocksTotal && errors.isEmpty() {
		wait.Add(1)
		go readFileExec(fileName, blockSize, processedBlocks, nrBlocksPerThread, &errors, &wait, hashes)
		processedBlocks += nrBlocksPerThread
	}
	wait.Wait()

	return hashes, errors.get(0)
}

func main() {
	fmt.Println("ok")
}
