package lacrima

func Eval(pos *Position, colour int8) int {
	score := 0
	for _, piece := range pos.Board {
		switch piece {
		case WhitePawn:
			score += 100
		case WhiteKnight:
			score += 320
		case WhiteBishop:
			score += 330
		case WhiteRook:
			score += 500
		case WhiteQueen:
			score += 900
		case BlackPawn:
			score -= 100
		case BlackKnight:
			score -= 320
		case BlackBishop:
			score -= 330
		case BlackRook:
			score -= 500
		case BlackQueen:
			score -= 900
		}
	}
	if colour < 0 {
		score = -score
	}
	return score
}
