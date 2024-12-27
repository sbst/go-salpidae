package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

const blockSize int64 = 1024 * 1024

func read(fileName string, startBlockId int64, nrBlocks int64, result []string) {
	f, e := os.Open(fileName)
	check(e)
	hasher := sha256.New()
	curBlockSize := blockSize
	block := make([]byte, curBlockSize)
	curBlockId := startBlockId
	for nrBlocks > 0 {
		n, e := f.Read(block)
		if n == 0 && e == io.EOF {
			break
		}
		check(e)
		if int64(n) < blockSize {
			curBlockSize = int64(n)
			block = block[0:curBlockSize]
		}
		if int64(n) > curBlockSize {
			panic("Unexpected read block size")
		}
		_, e = hasher.Write(block)
		check(e)
		sum := hasher.Sum(nil)
		result[curBlockId] = fmt.Sprintf("%x", sum)
		hasher.Reset()
		curBlockId += 1
		nrBlocks -= 1
	}
	e = f.Close()
	check(e)
}

func main() {
	fileName := "../dat2"
	f, e := os.Open(fileName)
	check(e)
	info, e := f.Stat()
	check(e)
	e = f.Close()
	check(e)
	nrBlocks := int64(math.Ceil(float64(info.Size()) / float64(blockSize)))
	hashes := make([]string, nrBlocks)
	read(fileName, 0, nrBlocks, hashes)
	for _, hash := range hashes {
		fmt.Println(hash)
	}
}
