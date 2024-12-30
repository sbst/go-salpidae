package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

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

type block struct {
	data []byte
	hash string
}
type blocks []block

func hashme(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func genBytes(item byte, size int) block {
	chunk := block{data: make([]byte, size)}
	for i := range chunk.data {
		chunk.data[i] = item
	}
	chunk.hash = hashme(chunk.data)
	return chunk
}

func genRandom(size int) block {
	chunk := block{data: make([]byte, size)}
	rand.Read(chunk.data)
	chunk.hash = hashme(chunk.data)
	return chunk
}

func writeBlocks(f *os.File, bs blocks) (int64, error) {
	var size int64 = 0
	for _, b := range bs {
		n, e := f.Write(b.data)
		if e != nil {
			return 0, e
		}
		size += int64(n)
	}
	return size, nil
}

func TestReadOneByOne(t *testing.T) {
	var blockSize int = 1024 * 1024

	zeroes := genBytes(0, blockSize)
	ones := genBytes(1, blockSize)

	f, e := os.CreateTemp("", "sal-tst")
	check(e)
	fileName := f.Name()
	defer os.Remove(fileName)
	writeBlocks(f, blocks{zeroes, ones, zeroes})

	expectedHashes := []string{
		zeroes.hash,
		ones.hash,
		zeroes.hash,
	}

	f, e = os.Open(fileName)
	check(e)
	nrBlocks := 3
	hashes := make([]string, nrBlocks)
	read(f, blockSize, 0, 1, hashes)
	read(f, blockSize, 1, 1, hashes)
	read(f, blockSize, 2, 1, hashes)
	e = f.Close()
	check(e)
	equal, msg := isEqual(hashes, expectedHashes)
	if !equal {
		t.Fatal(msg)
	}
}

type customReader struct {
	cb func() (int, error)
}

func (e customReader) Read(p []byte) (n int, err error) {
	return e.cb()
}

func TestReadError(t *testing.T) {
	var reader = customReader{
		cb: func() (int, error) { return 0, errors.New("nope") },
	}

	hashes := make([]string, 0)
	e := read(reader, 1, 0, 1, hashes)

	var expected *blockError
	if errors.As(e, &expected) {
		if expected.BlockId != 0 {
			t.Fatalf("Error in block: %v, expected: 0", expected.BlockId)
		}
	} else {
		t.Fatal("Error expected")
	}
}

func TestReadSmallBuffer(t *testing.T) {
	data := "abcde"
	var blockSize int = 1024 * 1024
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

func TestReadFilePrecise(t *testing.T) {
	var blockSize int = 1024 * 1024

	input := make(blocks, 6)
	for i := 0; i < 6; i++ {
		input[i] = genRandom(blockSize)
	}

	f, e := os.CreateTemp("", "sal-tst")
	check(e)
	fileName := f.Name()
	defer os.Remove(fileName)
	size, e := writeBlocks(f, input)
	check(e)

	expectedHashes := []string{}
	for _, item := range input {
		expectedHashes = append(expectedHashes, item.hash)
	}

	hashes, e := readFile(fileName, size, blockSize, 1)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}
	equal, msg := isEqual(hashes, expectedHashes)
	if !equal {
		t.Fatal(msg)
	}
}

func TestReadFileHalfblock(t *testing.T) {
	var blockSize int = 1024 * 1024

	input := make(blocks, 7)
	for i := range input {
		input[i] = genRandom(blockSize)
	}
	input[6] = genRandom(blockSize / 2)

	f, e := os.CreateTemp("", "sal-tst")
	check(e)
	fileName := f.Name()
	defer os.Remove(fileName)
	size, e := writeBlocks(f, input)
	check(e)

	expectedHashes := []string{}
	for _, item := range input {
		expectedHashes = append(expectedHashes, item.hash)
	}

	hashes, e := readFile(fileName, size, blockSize, 1)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}
	equal, msg := isEqual(hashes, expectedHashes)
	if !equal {
		t.Fatal(msg)
	}
}

func TestReadMissingFileExec(t *testing.T) {
	var errs errorList
	var wg sync.WaitGroup
	var result []string
	wg.Add(1)
	readFileExec("aa", 1, 1, 1, &errs, &wg, result)
	if errs.isEmpty() || errs.get(0) == nil {
		t.Fatalf("Error expected")
	}
}

func TestReadFileMultipleThreads(t *testing.T) {
	var blockSize int = 1024 * 1024

	input := make(blocks, 20)
	for i := range input {
		input[i] = genRandom(blockSize)
	}

	f, e := os.CreateTemp("", "sal-tst")
	check(e)
	fileName := f.Name()
	defer os.Remove(fileName)
	size, e := writeBlocks(f, input)
	check(e)

	hashes1, e := readFile(fileName, size, blockSize, 1)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}
	hashes2, e := readFile(fileName, size, blockSize, 100)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}
	equal, msg := isEqual(hashes1, hashes2)
	if !equal {
		t.Fatal(msg)
	}
}
