package salpidae

import (
	"math"
)

func GetNrOfBlocks(size int64, blockSize int) int {
	return int(math.Ceil(float64(size) / float64(blockSize)))
}
