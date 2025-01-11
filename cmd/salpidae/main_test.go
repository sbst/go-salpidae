package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
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

func (e customReader) ReadAt(p []byte, offset int64) (n int, err error) {
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

func TestReadCombinedSource(t *testing.T) {
	var blockSize int = 1
	const nrBlocksPerThreadConst = 3

	data := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9}

	bReader := bytes.NewReader(data)
	bResult, e := readFile(bReader, int64(len(data)), blockSize, nrBlocksPerThreadConst)
	check(e)

	input := make(blocks, 10)
	for i := 0; i < 10; i++ {
		input[i].data = []byte{data[i]}
		input[i].hash = hashme(input[i].data)
	}
	f, e := os.CreateTemp("", "sal-tst")
	check(e)
	fileName := f.Name()
	defer os.Remove(fileName)
	_, e = writeBlocks(f, input)
	f.Close()
	check(e)

	fReader, e := os.Open(fileName)
	check(e)
	defer fReader.Close()
	info, _ := fReader.Stat()

	fResult, e := readFile(bReader, info.Size(), blockSize, nrBlocksPerThreadConst)
	check(e)

	if len(input) != len(fResult) {
		t.Fatalf("Different result sizes: input - %v, buffer - %v", len(input), len(fResult))
	}
	if len(bResult) != len(fResult) {
		t.Fatalf("Different result sizes: buffer - %v, file - %v", len(bResult), len(fResult))
	}
	for i, _ := range input {
		if input[i].hash != bResult[i] {
			t.Fatalf("Different sum, block %v: input - %s, buffer - %s", i, input[i].hash, fResult[i])
		}
		if bResult[i] != fResult[i] {
			t.Fatalf("Different sum, block %v: buffer - %s, file - %s", i, bResult[i], fResult[i])
		}
	}
	// for i, _ := range bResult {
	// 	t.Logf("[%v]: %s", i, bResult[i])
	// }
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
	f.Close()
	check(e)

	expectedHashes := []string{}
	for _, item := range input {
		expectedHashes = append(expectedHashes, item.hash)
	}

	f, e = os.Open(fileName)
	check(e)
	defer f.Close()
	hashes, e := readFile(f, size, blockSize, 1)
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
	f.Close()

	expectedHashes := []string{}
	for _, item := range input {
		expectedHashes = append(expectedHashes, item.hash)
	}

	f, e = os.Open(fileName)
	check(e)
	defer f.Close()
	hashes, e := readFile(f, size, blockSize, 1)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}
	equal, msg := isEqual(hashes, expectedHashes)
	if !equal {
		t.Fatal(msg)
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
	f.Close()

	file1, e := os.Open(fileName)
	check(e)
	defer file1.Close()
	hashes1, e := readFile(file1, size, blockSize, 1)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}

	file2, e := os.Open(fileName)
	check(e)
	defer file2.Close()
	hashes2, e := readFile(file2, size, blockSize, 100)
	if e != nil {
		t.Fatalf("Error: %v", e.Error())
	}
	equal, msg := isEqual(hashes1, hashes2)
	if !equal {
		t.Fatal(msg)
	}
}
