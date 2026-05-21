package lacrima

import (
	"context"
	stdtime "time"
)

const mateScore = 100000
const infoInterval = stdtime.Second

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

func negamax(pos *Position, depth int, alpha int, beta int, colour int8, deadline stdtime.Time, ctx context.Context, ply int, nodes *uint64) (int, bool) {
	if searchStopped(ctx, deadline) {
		return 0, false
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

		score, ok := negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, ply+1, nodes)

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
	return getBestMoveWithInfo(ctx, pos, depth, time, nil)
}

func getBestMoveWithInfo(ctx context.Context, pos *Position, depth int, time int, onInfo SearchInfoFunc) Move {
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
	bestScore := -mateScore
	var nodes uint64
	lastInfoTime := startTime.Add(-infoInterval)

	if searchStopped(ctx, stdtime.Time{}) {
		return bestMove
	}

	alpha := -mateScore
	beta := mateScore

	for i, move := range moves {
		if searchStopped(ctx, deadline) {
			break
		}

		undo := MakeMove(pos, move)

		score, ok := negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, 1, &nodes)

		UnmakeMove(pos, undo)

		if !ok {
			break
		}

		score = -score

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

		if onInfo != nil {
			now := stdtime.Now()
			if i == 0 || now.Sub(lastInfoTime) >= infoInterval || i == len(moves)-1 {
				lastInfoTime = now
				onInfo(SearchInfo{
					Depth:            depth,
					Score:            bestScore,
					Nodes:            nodes,
					TimeMillis:       now.Sub(startTime).Milliseconds(),
					BestMove:         bestMove,
					CurrentMove:      move,
					CurrentMoveIndex: i + 1,
				})
			}
		}
	}

	return bestMove
}
