package lacrima

import "sort"

const checkMoveBonus = 1000
const preferredMoveBonus = 1000000

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
	undo := MakeMove(pos, move)
	kingSquare := FindKing(pos, pos.SideToMove)
	if InCheck(pos, pos.SideToMove, kingSquare) {
		score += checkMoveBonus
	}
	UnmakeMove(pos, undo)
	return score
}

func orderMoves(pos *Position, moves []Move, preferredMove Move) []Move {
	scoredMoves := make([]ScoredMove, len(moves))
	for i, move := range moves {
		score := ScoreMove(pos, move)
		if move == preferredMove {
			score += preferredMoveBonus
		}

		scoredMoves[i] = ScoredMove{
			Move:  move,
			Score: score,
		}
	}

	sort.SliceStable(scoredMoves, func(i, j int) bool {
		return scoredMoves[i].Score > scoredMoves[j].Score
	})

	ordered := make([]Move, len(scoredMoves))
	for i, scoredMove := range scoredMoves {
		ordered[i] = scoredMove.Move
	}

	return ordered
}
