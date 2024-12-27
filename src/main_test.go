package main

import (
	"math"
	"os"
	"testing"
)

func TestReadHalfBlockFullFile(t *testing.T) {
	fileName := "../../dat2"
	expected := [...]string{
		"bee2a0ac4cf8f4f77117d3e0be917cac8c600dfb99154018ef3cebe392724298",
		"0bf0567803b83fa4a4202d939365b7dde052af4086eadb19ca9494cb2cb28bf4",
		"a554c7573c4760d40b2d391557792d9f6588ee3200fef6ff6b34b7bbcb04659d",
		"a92f8625752fb897698b6bcfcdea7b5fb8ca7439dd1307c1c6c85a4de0ec388a",
		"cf745822a74607477bdb7e7a952941098fd09692c64b7c468d115b1c1a5fdace",
		"dcf2c240d59f12c3fda471fc54d4451184c90f08981556f61657b51453ab5f7a",
	}

	f, e := os.Open(fileName)
	check(e)
	info, e := f.Stat()
	check(e)
	e = f.Close()
	check(e)
	nrBlocks := int64(math.Ceil(float64(info.Size()) / float64(blockSize)))
	hashes := make([]string, nrBlocks)
	read(fileName, 0, nrBlocks, hashes)
	if len(hashes) != len(expected) {
		t.Fatalf("Hash size array: %v, expected: %v ", len(hashes), len(expected))
	}

	for i := range hashes {
		if hashes[i] != expected[i] {
			t.Fatalf("Hash for %v block: %v, expected: %v ", i, hashes[i], expected[i])
		}
	}
}

func TestReadFullBlockFullFile(t *testing.T) {
	fileName := "../../dat"
	expected := [...]string{
		"460960a020fbad248be7e4c93900bbe36bbd371c7bc61e426d3d7748e34acbdf",
		"075976462a537d4360fe64ddf3e1ef0dd7db14f003aecc630861b57c22bfa518",
		"e8f1614c60cb143a301b3fc3a9f039a7c10cca6c8a64cd6f52624fdf06970303",
	}

	f, e := os.Open(fileName)
	check(e)
	info, e := f.Stat()
	check(e)
	e = f.Close()
	check(e)
	nrBlocks := int64(math.Ceil(float64(info.Size()) / float64(blockSize)))
	hashes := make([]string, nrBlocks)
	read(fileName, 0, nrBlocks, hashes)
	if len(hashes) != len(expected) {
		t.Fatalf("Hash size array: %v, expected: %v ", len(hashes), len(expected))
	}

	for i := range hashes {
		if hashes[i] != expected[i] {
			t.Fatalf("Hash for %v block: %v, expected: %v ", i, hashes[i], expected[i])
		}
	}
}
