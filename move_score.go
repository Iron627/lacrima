package lacrima

const preferredMoveBonus = 1000000
const goodCaptureBonus = 800000
const promotionBonus = 700000
const killerMoveBonus = 600000
const badCapturePenalty = -800000
const maxHistoryValue = 8192
const historyScoreDivisor = 96
const contHistScoreDivisor = 96

var moveOrderingPieceValues = [6]int{100, 300, 325, 500, 900, 0}

func PieceValue(piece uint8) int {
	if piece >= uint8(len(moveOrderingPieceValues)) {
		return 0
	}
	return moveOrderingPieceValues[piece]
}

func ScoreMove(pos *Position, move Move) int {
	target := pos.PieceAt(move.To)
	if move.isEnPassant {
		target = Pawn
	}

	if target != Empty {
		attacker := pos.PieceAt(move.From)
		see := StaticExchangeEvaluation(pos, move)

		if see >= 0 {
			return goodCaptureBonus +
				PieceValue(target)*10 -
				PieceValue(attacker) +
				see
		}

		return badCapturePenalty + see
	}

	if isPromotion(move) {
		return promotionBonus + PieceValue(move.Promotion)
	}

	return 0
}

func scoreMoves(pos *Position, moves []Move, preferredMove Move, killerA Move, killerB Move, historyTable *[2][128][128]int, contHist *[2][6][64][6][64]int, previousMove Move, hasPreviousMove bool, scores []int) {
	side := 0
	if pos.SideToMove == Black {
		side = 1
	}

	var previousPiece uint8
	if hasPreviousMove {
		previousPiece = pos.PieceAt(previousMove.To)
	}

	for i, move := range moves {
		score := ScoreMove(pos, move)

		if move == preferredMove {
			score += preferredMoveBonus
		} else if isQuietMove(pos, move) {
			if move == killerA || move == killerB {
				score += killerMoveBonus
			}
			score += historyTable[side][move.From][move.To] / historyScoreDivisor
			if hasPreviousMove {
				piece := pos.PieceAt(move.From)
				score += contHist[side][previousPiece][previousMove.To][piece][move.To] / contHistScoreDivisor
			}
		}

		scores[i] = score
	}
}

func qScoreMove(pos *Position, move Move) int {
	target := pos.PieceAt(move.To)
	if move.isEnPassant {
		target = Pawn
	}

	if target != Empty {
		attacker := pos.PieceAt(move.From)
		see := StaticExchangeEvaluation(pos, move)

		if see >= 0 {
			return goodCaptureBonus +
				PieceValue(target)*10 -
				PieceValue(attacker) +
				see
		}

		return badCapturePenalty + see
	}

	if isPromotion(move) {
		return promotionBonus + PieceValue(move.Promotion)
	}

	return 0
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
