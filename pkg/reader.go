package salpidae

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
)

type BlockError struct {
	BlockId int
	Err     error
}

func (e *BlockError) Error() string {
	return fmt.Sprintf("Block %v error: %v", e.BlockId, e.Err.Error())
}

func (e *BlockError) Unwrap() error { return e.Err }

func read(reader io.ReaderAt, blockSize int, startBlock int, nrBlocks int, result []string) error {
	lastBlock := startBlock + nrBlocks
	hasher := sha256.New()
	for curBlock := startBlock; curBlock < lastBlock; curBlock++ {
		offset := curBlock * blockSize
		bSection := io.NewSectionReader(reader, int64(offset), int64(blockSize))

		n, e := io.Copy(hasher, bSection)
		if n == 0 {
			log.Printf("Block calculation problem\n")
			return &BlockError{curBlock, errors.New("zero copy")}
		}
		if e != nil {
			return &BlockError{curBlock, e}
		}
		result[curBlock] = fmt.Sprintf("%x", hasher.Sum(nil))
		hasher.Reset()
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

func ReadFile(reader io.ReaderAt, size int64, blockSize int, nrBlocksPerThread int) ([]string, error) {
	var wait = sync.WaitGroup{}
	var errors errorList

	totalBlocks := GetNrOfBlocks(size, blockSize)
	hashes := make([]string, totalBlocks)
	if nrBlocksPerThread > totalBlocks {
		nrBlocksPerThread = totalBlocks
		log.Printf("Too many blocks per thread for full file, limiting to %d\n", totalBlocks)
	}
	var processedBlocks int = 0
	for processedBlocks < totalBlocks && errors.isEmpty() {
		if processedBlocks+nrBlocksPerThread > totalBlocks {
			nrBlocksPerThread = totalBlocks - processedBlocks
			log.Printf("Too many blocks per thread for last block, limiting to remain blocks %d\n", totalBlocks-processedBlocks)
		}
		wait.Add(1)
		go func(startBlock int, nrBlocks int) {
			defer wait.Done()
			if e := read(reader, blockSize, startBlock, nrBlocks, hashes); e != nil {
				errors.add(e)
			}
		}(processedBlocks, nrBlocksPerThread)
		processedBlocks += nrBlocksPerThread
	}
	wait.Wait()
	return hashes, errors.get(0)
}
