package lacrima

import (
	"context"
	stdtime "time"
)

const mateScore = 100000

func searchStopped(ctx context.Context, deadline stdtime.Time) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}

	return !deadline.IsZero() && stdtime.Now().After(deadline)
}

func negamax(pos *Position, depth int, alpha int, beta int, colour int8, deadline stdtime.Time, ctx context.Context, ply int) (int, bool) {
	if searchStopped(ctx, deadline) {
		return 0, false
	}

	moves := GetLegalMoves(pos, colour)

	if len(moves) == 0 {
		if InCheck(pos, colour) {
			return -mateScore + ply, true
		}
		return 0, true
	}

	if depth <= 0 {
		return Eval(pos, colour), true
	}

	for _, move := range moves {
		undo := MakeMove(pos, move)

		score, ok := negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, ply+1)

		UnmakeMove(pos, undo)

		if !ok {
			return 0, false
		}

		score = -score

		if score >= beta {
			return beta, true
		}

		if score > alpha {
			alpha = score
		}
	}

	return alpha, true
}

func GetBestMove(pos *Position, depth int, time int) Move {
	return getBestMove(context.Background(), pos, depth, time)
}

func getBestMove(ctx context.Context, pos *Position, depth int, time int) Move {
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

	if searchStopped(ctx, stdtime.Time{}) {
		return bestMove
	}

	alpha := -mateScore
	beta := mateScore

	for _, move := range moves {
		if searchStopped(ctx, deadline) {
			break
		}

		undo := MakeMove(pos, move)

		score, ok := negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, 1)

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
