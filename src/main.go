package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getNrOfBlocks(size int64, blockSize int64) int64 {
	return int64(math.Ceil(float64(size) / float64(blockSize)))
}

func getFileSize(fileName string) int64 {
	f, e := os.Open(fileName)
	check(e)
	info, e := f.Stat()
	check(e)
	e = f.Close()
	check(e)
	return info.Size()
}

func skipBlocks(file *os.File, blockSize int64, nrOfBlocks int64) {
	file.Seek(blockSize*nrOfBlocks, io.SeekStart)
}

func read(reader io.Reader, blockSize int64, startBlockId int64, nrBlocks int64, result []string) {
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
		}
		check(e)
		_, e = hasher.Write(block)
		check(e)
		result[curBlockId] = fmt.Sprintf("%x", hasher.Sum(nil))
		hasher.Reset()
		curBlockId += 1
		nrBlocks -= 1
	}
}

func readFileExec(fileName string, blockSize int64, startBlockId int64, nrBlocks int64, wait *sync.WaitGroup, result []string) {
	f, e := os.Open(fileName)
	skipBlocks(f, blockSize, startBlockId)
	check(e)
	read(f, blockSize, startBlockId, nrBlocks, result)
	e = f.Close()
	check(e)
	wait.Done()
}

func readFile(fileName string, blockSize int64, nrBlocksPerThread int64) []string {
	var wait = sync.WaitGroup{}
	nrBlocksTotal := getNrOfBlocks(getFileSize(fileName), blockSize)
	hashes := make([]string, nrBlocksTotal)
	var processedBlocks int64 = 0
	for processedBlocks < nrBlocksTotal {
		wait.Add(1)
		go readFileExec(fileName, blockSize, processedBlocks, nrBlocksPerThread, &wait, hashes)
		processedBlocks += nrBlocksPerThread
	}
	wait.Wait()
	return hashes
}

func main() {
	fmt.Println("ok")
}
