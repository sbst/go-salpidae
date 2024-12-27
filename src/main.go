package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

const blockSize int64 = 1024 * 1024

func getNrOfBlocks(size int64) int64 {
	return int64(math.Ceil(float64(size) / float64(blockSize)))
}

func read(reader io.Reader, startBlockId int64, nrBlocks int64, result []string) {
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

func main() {
	fmt.Println("ok")
}
