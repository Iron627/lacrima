package lacrima

const preferredMoveBonus = 1000000
const killerMoveBonus = 900000
const maxHistoryValue = 8192
const historyScoreDivisor = 96

func ScoreMove(pos *Position, move Move) int {
	score := 0
	target := pos.Board[move.To]
	if move.isEnPassant {
		target = BlackPawn
		if pos.Board[move.From] < 0 {
			target = WhitePawn
		}
	}
	if target != Empty {
		victim := PieceValue(PieceType(target))
		attacker := PieceType(pos.Board[move.From])
		score += victim + (7 - int(attacker))
	}
	if move.Promotion != 0 {
		score += PieceValue(PieceType(move.Promotion))
	}

	return score
}

func scoreMoves(pos *Position, moves []Move, preferredMove Move, killerA Move, killerB Move, historyTable *[2][128][128]int, scores []int) {
	side := 0
	if pos.SideToMove < 0 {
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
	target := pos.Board[move.To]
	if move.isEnPassant {
		target = BlackPawn
		if pos.Board[move.From] < 0 {
			target = WhitePawn
		}
	}
	if target != Empty {
		victim := PieceValue(PieceType(target))
		attacker := PieceType(pos.Board[move.From])
		score += victim + (7 - int(attacker))
	}
	if move.Promotion != 0 {
		score += PieceValue(PieceType(move.Promotion))
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
