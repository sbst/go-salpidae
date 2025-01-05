package main

import (
	"crypto/sha256"
	"errors"
	"flag"
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

func read(reader io.ReaderAt, blockSize int, startBlock int, nrBlocks int, result []string) error {
	lastBlock := startBlock + nrBlocks
	hasher := sha256.New()
	for curBlock := startBlock; curBlock < lastBlock; curBlock++ {
		offset := curBlock * blockSize
		bSection := io.NewSectionReader(reader, int64(offset), int64(blockSize))

		n, e := io.Copy(hasher, bSection)
		if n == 0 {
			// log - block calculation problem, must not be reachable
			return &blockError{curBlock, errors.New("zero copy")}
		}
		if e != nil {
			return &blockError{curBlock, e}
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

func readFile(reader io.ReaderAt, size int64, blockSize int, nrBlocksPerThread int) ([]string, error) {
	var wait = sync.WaitGroup{}
	var errors errorList

	totalBlocks := getNrOfBlocks(size, blockSize)
	hashes := make([]string, totalBlocks)
	if nrBlocksPerThread > totalBlocks {
		nrBlocksPerThread = totalBlocks
		// log - too many blocks per thread for full file, limit to totalBlocks
	}
	var processedBlocks int = 0
	for processedBlocks < totalBlocks && errors.isEmpty() {
		if processedBlocks+nrBlocksPerThread > totalBlocks {
			nrBlocksPerThread = totalBlocks - processedBlocks
			// log - too much blocks per thread for last block, limit to remain blocks (totalBlocks - processedBlocks)
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

func writeFile(fileName string, signature []string) error {
	f, e := os.Create(fileName)
	if e != nil {
		return e
	}
	defer f.Close()

	for _, hash := range signature {
		_, e = f.WriteString(hash + "\n")
		if e != nil {
			return e
		}
	}
	return nil
}

func main() {
	const nrThreads int = 30

	var fileInput string
	flag.StringVar(&fileInput, "i", "", "file for signature generation")
	var fileOutput string
	flag.StringVar(&fileOutput, "o", "", "file for signature output")
	blockSizeM := flag.Int("b", 1, "size of block in MB")
	flag.Parse()

	if len(fileInput) == 0 {
		fmt.Fprintf(os.Stderr, "'-i' input file argument is missing\n")
		os.Exit(1)
	}
	if len(fileOutput) == 0 {
		fmt.Fprintf(os.Stderr, "'-o' output file argument is missing\n")
		os.Exit(1)
	}

	if *blockSizeM <= 0 || *blockSizeM > 2047 {
		fmt.Fprintf(os.Stderr, "Unsupported block size\n")
		os.Exit(1)
	}

	blockSize := int((*blockSizeM) * 1024 * 1024)
	fileSize, e := getFileSize(fileInput)
	if e != nil {
		fmt.Fprintf(os.Stderr, "Unable to read input file: %v\n", e.Error())
		os.Exit(1)
	}
	file, e := os.Open(fileInput)
	if e != nil {
		fmt.Fprintf(os.Stderr, "Unable to read input file: %v\n", e.Error())
		os.Exit(1)
	}
	defer file.Close()
	nrBlocksPerThread := (getNrOfBlocks(fileSize, blockSize) / nrThreads) + 1
	signature, e := readFile(file, fileSize, blockSize, nrBlocksPerThread)
	if e != nil {
		fmt.Fprintf(os.Stderr, "Unable to hash input file: %v\n", e.Error())
		os.Exit(1)
	}

	e = writeFile(fileOutput, signature)
	if e != nil {
		fmt.Fprintf(os.Stderr, "Unable to write output: %v\n", e.Error())
		os.Exit(1)
	}
}
