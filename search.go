package lacrima

import (
	"context"
	stdtime "time"
)

const mateScore = 100000
const repetitionAvoidanceScore = -1
const repetitionDrawScore = 0
const nullMoveReduction = 2
const maxSearchPly = 128
const aspirationWindow = 50

type SearchInfo struct {
	Depth      int
	Score      int
	Nodes      uint64
	TimeMillis int64
	BestMove   Move
}

type SearchInfoFunc func(SearchInfo)

type searchContext struct {
	ctx          context.Context
	deadline     stdtime.Time
	history      RepetitionHistory
	tt           *TranspositionTable
	historyTable *[2][128][128]int
	killers      [maxSearchPly][2]Move
	nodes        uint64
}

func newSearchContext(ctx context.Context, deadline stdtime.Time, history RepetitionHistory, tt *TranspositionTable, historyTable *[2][128][128]int) *searchContext {
	return &searchContext{
		ctx:          ctx,
		deadline:     deadline,
		history:      history,
		tt:           tt,
		historyTable: historyTable,
	}
}

func searchStopped(ctx context.Context, deadline stdtime.Time) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}

	return !deadline.IsZero() && stdtime.Now().After(deadline)
}

func searchContextStopped(search *searchContext) bool {
	return searchStopped(search.ctx, search.deadline)
}

func negamax(search *searchContext, pos *Position, depth int, alpha int, beta int, colour int8, ply int, allowNull bool, isCheckExtended bool) (int, bool) {

	if searchContextStopped(search) {
		return 0, false
	}

	if search.history != nil && search.history[positionKey(pos)] >= 3 {
		return repetitionDrawScore, true
	}

	search.nodes++
	kingSquare := FindKing(pos, colour)
	inCheck := InCheck(pos, colour, kingSquare)

	if depth <= 0 {
		if !isCheckExtended && inCheck {
			return negamax(search, pos, 1, alpha, beta, colour, ply, allowNull, true)
		}
		if !inCheck {
			return quiesce(search, pos, alpha, beta, colour, ply)
		}
		return Eval(pos, colour), true
	}

	key := positionKey(pos)
	originalAlpha := alpha
	var ttMove Move
	if entry, ok := search.tt.Probe(key); ok {
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

	if allowNull && depth >= 3 && !inCheck && hasNonPawnMaterial(pos, colour) {
		undo := MakeNullMove(pos)
		score, ok := negamax(search, pos, depth-1-nullMoveReduction, -beta, -beta+1, -colour, ply+1, false, isCheckExtended)
		score = -score
		UnmakeNullMove(pos, undo)

		if !ok {
			return 0, false
		}

		if score >= beta {
			search.tt.Store(key, depth, beta, TTLowerBound, Move{})
			return beta, true
		}
	}

	var pseudoBuf [256]Move
	var legalBuf [256]Move
	var scoreBuf [256]int
	moves := getLegalMovesInto(pos, colour, pseudoBuf[:0], legalBuf[:0])
	var killerA, killerB Move
	if ply >= 0 && ply < maxSearchPly {
		killerA = search.killers[ply][0]
		killerB = search.killers[ply][1]
	}

	if len(moves) == 0 {
		if inCheck {
			return -mateScore + ply, true
		}
		return 0, true
	}

	scores := scoreBuf[:len(moves)]
	scoreMoves(pos, moves, ttMove, killerA, killerB, search.historyTable, scores)

	bestMove := moves[0]
	bestScore := -mateScore

	for moveIndex := range moves {
		move := pickBestMove(moves, scores, moveIndex)
		undo := MakeMove(pos, move)

		repetitionKey, count := pushRepetition(search.history, pos)

		var score int
		var ok bool
		if count >= 3 {
			score = repetitionDrawScore
			ok = true
		} else {
			score, ok = negamax(search, pos, depth-1, -beta, -alpha, -colour, ply+1, true, isCheckExtended)
			score = -score
		}
		popRepetition(search.history, repetitionKey)

		UnmakeMove(pos, undo)

		if !ok {
			return 0, false
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}
		if score >= beta {
			if isQuietMove(pos, move) {
				side := 0
				if colour < 0 {
					side = 1
				}

				search.historyTable[side][move.From][move.To] += depth * depth
				if search.historyTable[side][move.From][move.To] > maxHistoryValue {
					search.historyTable[side][move.From][move.To] = maxHistoryValue
				}

				if ply >= 0 && ply < maxSearchPly {
					storeKiller(&search.killers, ply, move)
				}
			}

			search.tt.Store(key, depth, score, TTLowerBound, move)
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
	search.tt.Store(key, depth, bestScore, flag, bestMove)

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

func searchBestMove(ctx context.Context, pos *Position, depth int, time int, history RepetitionHistory, onInfo SearchInfoFunc, tt *TranspositionTable, historyTable *[2][128][128]int) Move {
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
	previousScore := 0
	var totalNodes uint64

	if searchStopped(ctx, stdtime.Time{}) {
		return bestMove
	}

	if depth < 1 {
		depth = 1
	}

	for currentDepth := 1; currentDepth <= depth; currentDepth++ {
		alpha := -mateScore
		beta := mateScore
		if currentDepth > 1 {
			alpha = previousScore - aspirationWindow
			if alpha < -mateScore {
				alpha = -mateScore
			}

			beta = previousScore + aspirationWindow
			if beta > mateScore {
				beta = mateScore
			}
		}

		var move Move
		var score int
		for {
			moveNodes := uint64(0)
			ok := false
			search := newSearchContext(ctx, deadline, history, tt, historyTable)
			move, score, moveNodes, ok = searchDepth(pos, currentDepth, colour, search, alpha, beta)
			totalNodes += moveNodes

			if !ok {
				return bestMove
			}

			if score <= alpha && alpha > -mateScore {
				alpha = -mateScore
				beta = mateScore
				continue
			}

			if score >= beta && beta < mateScore {
				alpha = -mateScore
				beta = mateScore
				continue
			}

			break
		}

		bestMove = move
		previousScore = score

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

func searchDepth(pos *Position, depth int, colour int8, search *searchContext, alpha int, beta int) (Move, int, uint64, bool) {
	var pseudoBuf [256]Move
	var legalBuf [256]Move
	var scoreBuf [256]int
	moves := getLegalMovesInto(pos, colour, pseudoBuf[:0], legalBuf[:0])
	if len(moves) == 0 {
		return Move{}, 0, 0, true
	}

	rootKey := positionKey(pos)
	var ttMove Move
	if entry, ok := search.tt.Probe(rootKey); ok {
		ttMove = entry.Move
	}

	scores := scoreBuf[:len(moves)]
	scoreMoves(pos, moves, ttMove, Move{}, Move{}, search.historyTable, scores)
	bestMove := moves[0]
	bestScore := -mateScore
	originalAlpha := alpha

	for moveIndex := range moves {
		if searchContextStopped(search) {
			return bestMove, bestScore, search.nodes, false
		}

		move := pickBestMove(moves, scores, moveIndex)
		undo := MakeMove(pos, move)

		var score int
		var ok bool
		repetitionKey, count := pushRepetition(search.history, pos)
		if count >= 3 {
			score = repetitionAvoidanceScore
			ok = true
		} else {
			score, ok = negamax(search, pos, depth-1, -beta, -alpha, -colour, 1, true, false)
			score = -score
		}
		popRepetition(search.history, repetitionKey)

		UnmakeMove(pos, undo)

		if !ok {
			return bestMove, bestScore, search.nodes, false
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score > alpha {
			alpha = score
		}

		if alpha >= beta {
			search.tt.Store(rootKey, depth, score, TTLowerBound, move)
			return bestMove, bestScore, search.nodes, true
		}
	}

	flag := TTExact
	if bestScore <= originalAlpha {
		flag = TTUpperBound
	}
	search.tt.Store(rootKey, depth, bestScore, flag, bestMove)

	return bestMove, bestScore, search.nodes, true
}

func isQuietMove(pos *Position, move Move) bool {
	if move.isEnPassant || move.Promotion != 0 || move.isCastling {
		return false
	}
	return pos.Board[move.To] == Empty
}

func quiesce(search *searchContext, pos *Position, alpha int, beta int, colour int8, ply int) (int, bool) {
	if searchContextStopped(search) {
		return 0, false
	}

	search.nodes++

	key := positionKey(pos)
	originalAlpha := alpha
	kingSquare := FindKing(pos, colour)
	inCheck := InCheck(pos, colour, kingSquare)

	var ttMove Move
	if entry, ok := search.tt.Probe(key); ok {
		ttMove = entry.Move
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

	if !inCheck {
		score := Eval(pos, colour)

		if score >= beta {
			search.tt.Store(key, 0, score, TTLowerBound, Move{})
			return score, true
		}
		if score > alpha {
			alpha = score
		}
	}

	var pseudoMoveBuffer [256]Move
	var legalMoveBuffer [256]Move
	var scoreBuffer [256]int
	var moves []Move
	if inCheck {
		moves = getLegalMovesInto(pos, colour, pseudoMoveBuffer[:0], legalMoveBuffer[:0])
		if len(moves) == 0 {
			score := -mateScore + ply
			search.tt.Store(key, 0, score, TTExact, Move{})
			return score, true
		}
	} else {
		moves = getLegalTacticalMovesInto(pos, colour, pseudoMoveBuffer[:0], legalMoveBuffer[:0])
	}
	scores := scoreBuffer[:len(moves)]
	qScoreMoves(pos, moves, ttMove, scores)

	bestMove := Move{}

	for moveIndex := range moves {
		move := pickBestMove(moves, scores, moveIndex)
		undo := MakeMove(pos, move)

		score, ok := quiesce(search, pos, -beta, -alpha, -colour, ply+1)
		score = -score

		UnmakeMove(pos, undo)

		if !ok {
			return 0, false
		}

		if score >= beta {
			search.tt.Store(key, 0, score, TTLowerBound, move)
			return score, true
		}

		if score > alpha {
			alpha = score
			bestMove = move
		}
	}

	flag := TTExact
	if alpha <= originalAlpha {
		flag = TTUpperBound
	}

	search.tt.Store(key, 0, alpha, flag, bestMove)

	return alpha, true
}

func isTacticalMove(pos *Position, move Move) bool {
	if move.isEnPassant || move.Promotion != 0 {
		return true
	}
	return pos.Board[move.To] != Empty
}

func filterTacticalMoves(pos *Position, moves []Move) []Move {
	tacticalMoves := moves[:0]
	for _, move := range moves {
		if isTacticalMove(pos, move) {
			tacticalMoves = append(tacticalMoves, move)
		}
	}
	return tacticalMoves
}

func getLegalTacticalMovesInto(pos *Position, colour int8, pseudoMoves []Move, legalMoves []Move) []Move {
	originalSideToMove := pos.SideToMove
	defer func() {
		pos.SideToMove = originalSideToMove
	}()

	pos.SideToMove = colour
	pseudoMoves = GeneratePseudoLegalMovesInto(pos, pseudoMoves)
	pseudoMoves = filterTacticalMoves(pos, pseudoMoves)
	return filterLegalMovesInto(pos, pseudoMoves, legalMoves)
}

func storeKiller(killers *[maxSearchPly][2]Move, ply int, move Move) {
	if killers[ply][0] == move {
		return
	}
	killers[ply][1] = killers[ply][0]
	killers[ply][0] = move
}
