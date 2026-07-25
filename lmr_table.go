package lacrima

import "math"

const (
	lmrBias  = 0.8
	lmrScale = 2.5
)

func lmReduction(depth, moveIndex int) uint8 {
	moveIndex++
	return uint8(lmrBias + math.Log(float64(depth))*math.Log(float64(moveIndex))/lmrScale)
}

var LMRTable [129][256]uint8

func init() {
	for depth := 1; depth <= 128; depth++ {
		for moveIndex := 0; moveIndex < 256; moveIndex++ {
			LMRTable[depth][moveIndex] = lmReduction(depth, moveIndex)
		}
	}
}
