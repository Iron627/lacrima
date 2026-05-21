package lacrima

import (
	"context"
	stdtime "time"
)

const mateScore = 100000
const repetitionAvoidanceScore = -1
const repetitionDrawScore = 0

type SearchInfo struct {
	Depth            int
	Score            int
	Nodes            uint64
	TimeMillis       int64
	BestMove         Move
	CurrentMove      Move
	CurrentMoveIndex int
}

type SearchInfoFunc func(SearchInfo)

func searchStopped(ctx context.Context, deadline stdtime.Time) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}

	return !deadline.IsZero() && stdtime.Now().After(deadline)
}

func negamax(pos *Position, depth int, alpha int, beta int, colour int8, deadline stdtime.Time, ctx context.Context, ply int, nodes *uint64, history RepetitionHistory) (int, bool) {
	if searchStopped(ctx, deadline) {
		return 0, false
	}

	if history != nil && history[positionKey(pos)] >= 3 {
		return repetitionDrawScore, true
	}

	*nodes += 1

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

		key, count := pushRepetition(history, pos)

		var score int
		var ok bool
		if count >= 3 {
			score = repetitionDrawScore
			ok = true
		} else {
			score, ok = negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, ply+1, nodes, history)
			score = -score
		}
		popRepetition(history, key)

		UnmakeMove(pos, undo)

		if !ok {
			return 0, false
		}

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
	return getBestMoveWithInfo(ctx, pos, depth, time, nil, nil)
}

func getBestMoveWithInfo(ctx context.Context, pos *Position, depth int, time int, history RepetitionHistory, onInfo SearchInfoFunc) Move {
	colour := pos.SideToMove
	originalSideToMove := pos.SideToMove
	startTime := stdtime.Now()

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
	var totalNodes uint64

	if searchStopped(ctx, stdtime.Time{}) {
		return bestMove
	}

	if depth < 1 {
		depth = 1
	}

	for currentDepth := 1; currentDepth <= depth; currentDepth++ {
		move, score, nodes, ok := searchDepth(ctx, pos, currentDepth, deadline, colour, history)
		totalNodes += nodes

		if !ok {
			break
		}

		bestMove = move

		if onInfo != nil {
			onInfo(SearchInfo{
				Depth:      currentDepth,
				Score:      score,
				Nodes:      totalNodes,
				TimeMillis: stdtime.Since(startTime).Milliseconds(),
				BestMove:   bestMove,
			})
		}

		if searchStopped(ctx, deadline) {
			break
		}
	}

	return bestMove
}

func searchDepth(ctx context.Context, pos *Position, depth int, deadline stdtime.Time, colour int8, history RepetitionHistory) (Move, int, uint64, bool) {
	moves := GetLegalMoves(pos, colour)
	if len(moves) == 0 {
		return Move{}, 0, 0, true
	}

	bestMove := moves[0]
	bestScore := -mateScore
	var nodes uint64
	alpha := -mateScore
	beta := mateScore

	for _, move := range moves {
		if searchStopped(ctx, deadline) {
			return bestMove, bestScore, nodes, false
		}

		undo := MakeMove(pos, move)

		var score int
		var ok bool
		key, count := pushRepetition(history, pos)
		if count >= 3 {
			score = repetitionAvoidanceScore
			ok = true
		} else {
			score, ok = negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, 1, &nodes, history)
			score = -score
		}
		popRepetition(history, key)

		UnmakeMove(pos, undo)

		if !ok {
			return bestMove, bestScore, nodes, false
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score > alpha {
			alpha = score
		}

		if alpha >= beta {
			break
		}
	}

	return bestMove, bestScore, nodes, true
}
