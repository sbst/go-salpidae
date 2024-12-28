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
	BlockId int64
	Err     error
}

func (e *blockError) Error() string {
	return fmt.Sprintf("Block %v error: %v", e.BlockId, e.Err.Error())
}

func (e *blockError) Unwrap() error { return e.Err }

func getNrOfBlocks(size int64, blockSize int64) int64 {
	return int64(math.Ceil(float64(size) / float64(blockSize)))
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

func skipBlocks(file *os.File, blockSize int64, nrOfBlocks int64) error {
	_, e := file.Seek(blockSize*nrOfBlocks, io.SeekStart)
	return e
}

func read(reader io.Reader, blockSize int64, startBlockId int64, nrBlocks int64, result []string) error {
	hasher := sha256.New()
	block := make([]byte, blockSize)
	curBlockId := startBlockId
	for nrBlocks > 0 {
		n, e := io.ReadFull(reader, block)
		if n == 0 && e == io.EOF {
			break
		} else if e == io.ErrUnexpectedEOF {
			block = block[0:int64(n)]
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

func readFileExec(fileName string, blockSize int64, startBlockId int64, nrBlocks int64, wait *sync.WaitGroup, result []string) {
	defer wait.Done()
	f, e := os.Open(fileName)
	if e != nil {
		return
	}
	defer f.Close()

	e = skipBlocks(f, blockSize, startBlockId)
	if e != nil {
		return
	}

	e = read(f, blockSize, startBlockId, nrBlocks, result)
	if e != nil {
		return
	}
}

func readFile(fileName string, blockSize int64, nrBlocksPerThread int64) ([]string, error) {
	var wait = sync.WaitGroup{}
	fileSize, e := getFileSize(fileName)
	if e != nil {
		return make([]string, 0), e
	}
	nrBlocksTotal := getNrOfBlocks(fileSize, blockSize)
	hashes := make([]string, nrBlocksTotal)
	var processedBlocks int64 = 0
	for processedBlocks < nrBlocksTotal {
		wait.Add(1)
		go readFileExec(fileName, blockSize, processedBlocks, nrBlocksPerThread, &wait, hashes)
		processedBlocks += nrBlocksPerThread
	}
	wait.Wait()
	return hashes, nil
}

func main() {
	fmt.Println("ok")
}
