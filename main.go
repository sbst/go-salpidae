package main

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	f, e := os.Open("../dat")
	check(e)
	block := make([]byte, 1024*1024)
	_, e = f.Read(block)
	check(e)
	hasher := sha256.New()
	_, e = hasher.Write(block)
	check(e)
	sum := hasher.Sum(nil)
	fmt.Printf("%x\n", sum)
}
