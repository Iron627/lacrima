package lacrima

import stdtime "time"

const mateScore = 100000

func searchTimedOut(deadline stdtime.Time) bool {
	return !deadline.IsZero() && stdtime.Now().After(deadline)
}

func negamax(pos *Position, depth int, alpha int, beta int, colour int8, deadline stdtime.Time) (int, bool) {
	if searchTimedOut(deadline) {
		return Eval(pos, colour), false
	}

	moves := GetLegalMoves(pos, colour)
	if depth <= 0 || len(moves) == 0 {
		return Eval(pos, colour), true
	}

	for _, move := range moves {
		undo := MakeMove(pos, move)
		score, ok := negamax(pos, depth-1, -beta, -alpha, -colour, deadline)
		UnmakeMove(pos, undo)
		if !ok {
			return alpha, false
		}
		score = -score
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			break
		}
	}

	return alpha, true
}

func GetBestMove(pos *Position, depth int, time int) Move {
	colour := pos.SideToMove
	originalSideToMove := pos.SideToMove
	defer func() {
		pos.SideToMove = originalSideToMove
	}()

	var deadline stdtime.Time
	if time > 0 {
		deadline = stdtime.Now().Add(stdtime.Duration(time) * stdtime.Millisecond)
	}

	moves := GetLegalMoves(pos, colour)
	if len(moves) == 0 {
		return Move{}
	}

	bestMove := moves[0]
	alpha := -mateScore
	for _, move := range moves {
		if searchTimedOut(deadline) {
			break
		}
		undo := MakeMove(pos, move)
		score, ok := negamax(pos, depth-1, -mateScore, mateScore, -colour, deadline)
		UnmakeMove(pos, undo)
		if !ok {
			break
		}
		score = -score
		if score > alpha {
			alpha = score
			bestMove = move
		}
	}
	return bestMove
}
