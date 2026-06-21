package lacrima

func Eval(pos *Position) int {
	mg := [2]int{}
	eg := [2]int{}
	gamePhase := 0
	pawns := make([]int, 0, 16)
	bishopCount := [2]int{}
	for sq, piece := range pos.Board {
		if IsOffBoard(sq) || piece == Empty {
			continue
		}

		side := 0
		if piece < 0 {
			side = 1
		}

		pt := int(PieceType(piece)) - 1
		if pt == 0 {
			pawns = append(pawns, sq)
		}
		if pt == 2 {
			bishopCount[side]++
		}
		pstSq := sq64(sq)

		if side == 1 {
			pstSq ^= 56
		}

		mg[side] += MgValue[pt] + MgPestoTable[pt][pstSq]
		eg[side] += EgValue[pt] + EgPestoTable[pt][pstSq]

		gamePhase += GamePhaseInc[pt]
	}
	for side, count := range bishopCount {
		if count >= 2 {
			mg[side] += BishopPairMgBonus
			eg[side] += BishopPairEgBonus
		}
	}

	for _, sq := range pawns {
		if !isPassed(pos, sq, pawns) {
			continue
		}

		piece := pos.Board[sq]
		side := 0
		advance := sq >> 4
		if piece < 0 {
			side = 1
			advance = 7 - (sq >> 4)
		}

		mg[side] += PassedPawnMgBonus[advance]
		eg[side] += PassedPawnEgBonus[advance]
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

func isPassed(pos *Position, sq int, pawns []int) bool {
	pawn := pos.Board[sq]
	if PieceType(pawn) != 1 {
		return false
	}

	for _, p := range pawns {
		if p == sq {
			continue
		}

		fileDistance := (p & 7) - (sq & 7)
		if fileDistance < 0 {
			fileDistance = -fileDistance
		}
		if fileDistance > 1 {
			continue
		}

		isAhead := pawn > 0 && p>>4 > sq>>4 ||
			pawn < 0 && p>>4 < sq>>4
		if !isAhead {
			continue
		}

		otherPawn := pos.Board[p]
		if p&7 == sq&7 || !SameColor(pawn, otherPawn) {
			return false
		}
	}

	return true
}
