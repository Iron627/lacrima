package lacrima

import "math/bits"

var passedPawnMasks = initPassedPawnMasks()

func Eval(pos *Position) int {
	mg := [2]int{}
	eg := [2]int{}
	gamePhase := 0
	bishopCount := [2]int{
		bits.OnesCount64(uint64(pos.Board.GetPieceBoard(White, Bishop))),
		bits.OnesCount64(uint64(pos.Board.GetPieceBoard(Black, Bishop))),
	}
	for side := 0; side < 2; side++ {
		for pt := Pawn; pt <= King; pt++ {
			pieces := pos.Board.GetPieceBoard(uint8(side), uint8(pt))
			for pieces != 0 {
				sq := popLSB(&pieces)
				pstSq := sq64(sq)
				if side == Black {
					pstSq ^= 56
				}

				mg[side] += MgValue[pt] + MgPestoTable[pt][pstSq]
				eg[side] += EgValue[pt] + EgPestoTable[pt][pstSq]
				gamePhase += GamePhaseInc[pt]
			}
		}
	}
	for side, count := range bishopCount {
		if count >= 2 {
			mg[side] += BishopPairMgBonus
			eg[side] += BishopPairEgBonus
		}
	}

	for side := 0; side < 2; side++ {
		pawns := pos.Board.GetPieceBoard(uint8(side), Pawn)
		enemyPawns := pos.Board.GetPieceBoard(uint8(side^1), Pawn)
		for pawns != 0 {
			sq := popLSB(&pawns)
			if enemyPawns&passedPawnMasks[side][sq] != 0 {
				continue
			}

			advance := sq / 8
			if side == Black {
				advance = 7 - advance
			}

			mg[side] += PassedPawnMgBonus[advance]
			eg[side] += PassedPawnEgBonus[advance]
		}
	}

	if gamePhase > 24 {
		gamePhase = 24
	}

	mgScore := mg[0] - mg[1]
	egScore := eg[0] - eg[1]

	egPhase := 24 - gamePhase

	score := (mgScore*gamePhase +
		egScore*egPhase) / 24

	if pos.SideToMove == Black {
		score = -score
	}

	return score
}

func sq64(sq int) int {
	rank := sq / 8
	file := sq & 7

	return (7-rank)*8 + file
}

func initPassedPawnMasks() [2][64]Bitboard {
	var masks [2][64]Bitboard
	for sq := 0; sq < 64; sq++ {
		rank := sq / 8
		file := sq & 7
		for _, df := range [...]int{-1, 0, 1} {
			f := file + df
			if f < 0 || f >= 8 {
				continue
			}
			for r := rank + 1; r < 8; r++ {
				masks[White][sq] |= Bitboard(1) << uint(r*8+f)
			}
			for r := rank - 1; r >= 0; r-- {
				masks[Black][sq] |= Bitboard(1) << uint(r*8+f)
			}
		}
	}
	return masks
}
