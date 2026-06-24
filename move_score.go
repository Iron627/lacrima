package lacrima

const preferredMoveBonus = 1000000
const killerMoveBonus = 900000
const maxHistoryValue = 8192
const historyScoreDivisor = 96

var moveOrderingPieceValues = [6]int{100, 300, 325, 500, 900, 0}

func PieceValue(piece uint8) int {
	if piece >= uint8(len(moveOrderingPieceValues)) {
		return 0
	}
	return moveOrderingPieceValues[piece]
}

func ScoreMove(pos *Position, move Move) int {
	score := 0
	target := pos.PieceAt(move.To)
	if move.isEnPassant {
		target = Pawn
	}
	if target != Empty {
		victim := PieceValue(target)
		attacker := pos.PieceAt(move.From)
		score += victim + (7 - int(attacker))
	}
	if isPromotion(move) {
		score += PieceValue(move.Promotion)
	}

	return score
}

func scoreMoves(pos *Position, moves []Move, preferredMove Move, killerA Move, killerB Move, historyTable *[2][128][128]int, scores []int) {
	side := 0
	if pos.SideToMove == Black {
		side = 1
	}

	for i, move := range moves {
		score := ScoreMove(pos, move)
		quiet := isQuietMove(pos, move)
		if move == preferredMove {
			score += preferredMoveBonus
		} else if move == killerA || move == killerB {
			score += killerMoveBonus
		}
		if quiet {
			score += historyTable[side][move.From][move.To] / historyScoreDivisor
		}
		scores[i] = score
	}
}

func qScoreMove(pos *Position, move Move) int {
	score := 0
	target := pos.PieceAt(move.To)
	if move.isEnPassant {
		target = Pawn
	}
	if target != Empty {
		victim := PieceValue(target)
		attacker := pos.PieceAt(move.From)
		score += victim + (7 - int(attacker))
	}
	if isPromotion(move) {
		score += PieceValue(move.Promotion)
	}
	return score
}

func qScoreMoves(pos *Position, moves []Move, preferredMove Move, scores []int) {
	for i, move := range moves {
		score := qScoreMove(pos, move)
		if move == preferredMove {
			score += preferredMoveBonus
		}

		scores[i] = score
	}
}

func pickBestMove(moves []Move, scores []int, start int) Move {
	best := start
	for i := start + 1; i < len(moves); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}

	moves[start], moves[best] = moves[best], moves[start]
	scores[start], scores[best] = scores[best], scores[start]
	return moves[start]
}
