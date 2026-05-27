package lacrima

import (
	"context"
	stdtime "time"
)

const mateScore = 100000
const repetitionAvoidanceScore = -1
const repetitionDrawScore = 0
const defaultTranspositionTableEntries = 1 << 18
const nullMoveReduction = 2
const maxSearchPly = 128

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

func negamax(pos *Position, depth int, alpha int, beta int, colour int8, deadline stdtime.Time, ctx context.Context, ply int, nodes *uint64, history RepetitionHistory, tt *TranspositionTable, allowNull bool, killers *[maxSearchPly][2]Move) (int, bool) {
	if searchStopped(ctx, deadline) {
		return 0, false
	}

	if history != nil && history[positionKey(pos)] >= 3 {
		return repetitionDrawScore, true
	}

	*nodes += 1

	if depth <= 0 {
		return Eval(pos, colour), true
	}

	key := positionKey(pos)
	originalAlpha := alpha
	var ttMove Move
	if entry, ok := tt.Probe(key); ok {
		ttMove = entry.Move
		if entry.Depth >= depth {
			switch entry.Flag {
			case TTExact:
				return entry.Score, true
			case TTLowerBound:
				if entry.Score > alpha {
					alpha = entry.Score
				}
			case TTUpperBound:
				if entry.Score < beta {
					beta = entry.Score
				}
			}

			if alpha >= beta {
				return entry.Score, true
			}
		}
	}

	kingSquare := FindKing(pos, colour)
	inCheck := InCheck(pos, colour, kingSquare)
	if allowNull && depth >= 3 && !inCheck && hasNonPawnMaterial(pos, colour) {
		undo := MakeNullMove(pos)
		score, ok := negamax(pos, depth-1-nullMoveReduction, -beta, -beta+1, -colour, deadline, ctx, ply+1, nodes, history, tt, false, killers)
		score = -score
		UnmakeNullMove(pos, undo)

		if !ok {
			return 0, false
		}
		if score >= beta {
			tt.Store(key, depth, beta, TTLowerBound, Move{})
			return beta, true
		}
	}

	var pseudoBuf [256]Move
	var legalBuf [256]Move
	moves := getLegalMovesInto(pos, colour, pseudoBuf[:0], legalBuf[:0])
	var killerA, killerB Move
	if ply >= 0 && ply < maxSearchPly {
		killerA = killers[ply][0]
		killerB = killers[ply][1]
	}
	moves = orderMoves(pos, moves, ttMove, killerA, killerB)

	if len(moves) == 0 {
		if inCheck {
			return -mateScore + ply, true
		}
		return 0, true
	}

	bestMove := moves[0]
	bestScore := -mateScore

	for _, move := range moves {
		undo := MakeMove(pos, move)

		repetitionKey, count := pushRepetition(history, pos)

		var score int
		var ok bool
		if count >= 3 {
			score = repetitionDrawScore
			ok = true
		} else {
			score, ok = negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, ply+1, nodes, history, tt, true, killers)
			score = -score
		}
		popRepetition(history, repetitionKey)

		UnmakeMove(pos, undo)

		if !ok {
			return 0, false
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score >= beta {
			if ply >= 0 && ply < maxSearchPly && isQuietMove(pos, move) {
				storeKiller(killers, ply, move)
			}
			tt.Store(key, depth, score, TTLowerBound, move)
			return score, true
		}

		if score > alpha {
			alpha = score
		}
	}

	flag := TTExact
	if bestScore <= originalAlpha {
		flag = TTUpperBound
	}
	tt.Store(key, depth, bestScore, flag, bestMove)

	return bestScore, true
}

func hasNonPawnMaterial(pos *Position, colour int8) bool {
	for square, piece := range pos.Board {
		if IsOffBoard(square) || piece == Empty || (piece > 0) != (colour > 0) {
			continue
		}

		pieceType := PieceType(piece)
		if pieceType != WhitePawn && pieceType != WhiteKing {
			return true
		}
	}

	return false
}

func GetBestMove(pos *Position, depth int, time int) Move {
	return getBestMove(context.Background(), pos, depth, time)
}

func getBestMove(ctx context.Context, pos *Position, depth int, time int) Move {
	return searchBestMove(ctx, pos, depth, time, nil, nil, NewTranspositionTable(defaultTranspositionTableEntries))
}

func searchBestMove(ctx context.Context, pos *Position, depth int, time int, history RepetitionHistory, onInfo SearchInfoFunc, tt *TranspositionTable) Move {
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

	var pseudoBuf [256]Move
	var legalBuf [256]Move
	moves := getLegalMovesInto(pos, colour, pseudoBuf[:0], legalBuf[:0])
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
		move, score, nodes, ok := searchDepth(ctx, pos, currentDepth, deadline, colour, history, tt)
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

func searchDepth(ctx context.Context, pos *Position, depth int, deadline stdtime.Time, colour int8, history RepetitionHistory, tt *TranspositionTable) (Move, int, uint64, bool) {
	var killers [maxSearchPly][2]Move
	var pseudoBuf [256]Move
	var legalBuf [256]Move
	moves := getLegalMovesInto(pos, colour, pseudoBuf[:0], legalBuf[:0])
	if len(moves) == 0 {
		return Move{}, 0, 0, true
	}

	rootKey := positionKey(pos)
	var ttMove Move
	if entry, ok := tt.Probe(rootKey); ok {
		ttMove = entry.Move
	}

	moves = orderMoves(pos, moves, ttMove, Move{}, Move{})
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
		repetitionKey, count := pushRepetition(history, pos)
		if count >= 3 {
			score = repetitionAvoidanceScore
			ok = true
		} else {
			score, ok = negamax(pos, depth-1, -beta, -alpha, -colour, deadline, ctx, 1, &nodes, history, tt, true, &killers)
			score = -score
		}
		popRepetition(history, repetitionKey)

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
			tt.Store(rootKey, depth, score, TTLowerBound, move)
			break
		}
	}

	tt.Store(rootKey, depth, bestScore, TTExact, bestMove)

	return bestMove, bestScore, nodes, true
}

func isQuietMove(pos *Position, move Move) bool {
	if move.isEnPassant || move.Promotion != 0 || move.isCastling {
		return false
	}
	return pos.Board[move.To] == Empty
}

func storeKiller(killers *[maxSearchPly][2]Move, ply int, move Move) {
	if killers[ply][0] == move {
		return
	}
	killers[ply][1] = killers[ply][0]
	killers[ply][0] = move
}
