package lacrima

import "sort"

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
		score += PieceValue(PieceType(target))
	}
	if move.Promotion != 0 {
		score += PieceValue(PieceType(move.Promotion))
	}
	undo := MakeMove(pos, move)
	if InCheck(pos, pos.SideToMove) {
		score += 1000
	}
	UnmakeMove(pos, undo)
	return score
}

func orderMoves(pos *Position, moves []Move) []Move {
	scoredMoves := make([]ScoredMove, len(moves))
	for i, move := range moves {
		scoredMoves[i] = ScoredMove{
			Move:  move,
			Score: ScoreMove(pos, move),
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
