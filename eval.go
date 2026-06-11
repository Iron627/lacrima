package lacrima

func Eval(pos *Position) int {
	mg := [2]int{}
	eg := [2]int{}
	gamePhase := 0

	for sq, piece := range pos.Board {
		if IsOffBoard(sq) || piece == Empty {
			continue
		}

		side := 0
		if piece < 0 {
			side = 1
		}

		pt := int(PieceType(piece)) - 1

		pstSq := sq64(sq)

		if side == 1 {
			pstSq ^= 56
		}

		mg[side] += MgValue[pt] + MgPestoTable[pt][pstSq]
		eg[side] += EgValue[pt] + EgPestoTable[pt][pstSq]

		gamePhase += GamePhaseInc[pt]
	}

	if gamePhase > 24 {
		gamePhase = 24
	}

	mgScore := mg[0] - mg[1]
	egScore := eg[0] - eg[1]

	egPhase := 24 - gamePhase

	score := (mgScore*gamePhase +
		egScore*egPhase) / 24

	if pos.SideToMove < 0 {
		score = -score
	}

	return score
}

func sq64(sq int) int {
	rank := sq >> 4
	file := sq & 7

	return (7-rank)*8 + file
}
