package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// func readLoad(nrOfBlocksPerThread int64) string {
// 	numThreads = 0
// 	fileName := "../../datbig"
// 	start := time.Now()
// 	hashes := readFile(fileName, 1024, nrOfBlocksPerThread)
// 	elapsed := time.Since(start)
// 	createdThreads := numThreads
// 	numThreads = 0
// 	return fmt.Sprintf("\nNumber of blocks: %v\nBlocks per thread: %v\nCreated threads: %v\nElapsed: %.3fs\n\n", len(hashes), nrOfBlocksPerThread, createdThreads, elapsed.Seconds())
// }

// func TestLoad(t *testing.T) {
// 	var nrBlocks int64 = 512000 * 2
// 	var msg string
// 	for nrBlocks != 1 {
// 		msg += readLoad(nrBlocks)
// 		nrBlocks = nrBlocks / 2
// 	}
// 	t.Log(msg)
// }

func isEqual(hashes1 []string, hashes2 []string) (bool, string) {
	if len(hashes1) != len(hashes2) {
		return false, fmt.Sprintf("Hash size array: %v, expected: %v ", len(hashes1), len(hashes2))
	}

	for i := range hashes1 {
		if hashes1[i] != hashes2[i] {
			return false, fmt.Sprintf("Hash for %v block: %v, expected: %v ", i, hashes1[i], hashes2[i])
		}
	}
	return true, ""
}

func TestReadOneByOne(t *testing.T) {
	fileName := "../../dat2"
	var blockSize int64 = 1024 * 1024
	expected := []string{
		"bee2a0ac4cf8f4f77117d3e0be917cac8c600dfb99154018ef3cebe392724298",
		"0bf0567803b83fa4a4202d939365b7dde052af4086eadb19ca9494cb2cb28bf4",
		"a554c7573c4760d40b2d391557792d9f6588ee3200fef6ff6b34b7bbcb04659d",
	}

	f, e := os.Open(fileName)
	check(e)
	nrBlocks := 3
	hashes := make([]string, nrBlocks)
	read(f, blockSize, 0, 1, hashes)
	read(f, blockSize, 1, 1, hashes)
	read(f, blockSize, 2, 1, hashes)
	e = f.Close()
	check(e)
	equal, msg := isEqual(hashes, expected)
	if !equal {
		t.Fatal(msg)
	}
}

func TestReadSmallBuffer(t *testing.T) {
	data := "abcde"
	var blockSize int64 = 1024 * 1024
	totalBlocks := getNrOfBlocks(int64(len(data)), blockSize)
	expected := []string{
		"36bbe50ed96841d10443bcb670d6554f0a34b761be67ec9c4a8ad2c0c44ca42c",
	}

	hashes := make([]string, totalBlocks)
	reader := strings.NewReader("abcde")
	read(reader, blockSize, 0, totalBlocks, hashes)
	equal, msg := isEqual(hashes, expected)
	if !equal {
		t.Fatal(msg)
	}
}

func TestReadFile(t *testing.T) {
	fileName := "../../dat2"
	var blockSize int64 = 1024 * 1024
	expected := []string{
		"bee2a0ac4cf8f4f77117d3e0be917cac8c600dfb99154018ef3cebe392724298",
		"0bf0567803b83fa4a4202d939365b7dde052af4086eadb19ca9494cb2cb28bf4",
		"a554c7573c4760d40b2d391557792d9f6588ee3200fef6ff6b34b7bbcb04659d",
		"a92f8625752fb897698b6bcfcdea7b5fb8ca7439dd1307c1c6c85a4de0ec388a",
		"cf745822a74607477bdb7e7a952941098fd09692c64b7c468d115b1c1a5fdace",
		"dcf2c240d59f12c3fda471fc54d4451184c90f08981556f61657b51453ab5f7a",
	}
	hashes := readFile(fileName, blockSize, 2)
	equal, msg := isEqual(hashes, expected)
	if !equal {
		t.Fatal(msg)
	}
}

func TestReadFileFullBlockFullFile(t *testing.T) {
	fileName := "../../dat"
	var blockSize int64 = 1024 * 1024
	expected := []string{
		"460960a020fbad248be7e4c93900bbe36bbd371c7bc61e426d3d7748e34acbdf",
		"075976462a537d4360fe64ddf3e1ef0dd7db14f003aecc630861b57c22bfa518",
		"e8f1614c60cb143a301b3fc3a9f039a7c10cca6c8a64cd6f52624fdf06970303",
	}

	hashes := readFile(fileName, blockSize, 1)
	equal, msg := isEqual(hashes, expected)
	if !equal {
		t.Fatal(msg)
	}
}

func TestReadFileMultipleThreads(t *testing.T) {
	fileName := "../../dat"
	var blockSize int64 = 1024 * 1024

	hashes1 := readFile(fileName, blockSize, 1)
	hashes2 := readFile(fileName, blockSize, 100)
	equal, msg := isEqual(hashes1, hashes2)
	if !equal {
		t.Fatal(msg)
	}
}
