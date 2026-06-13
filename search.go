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
const LMReduction = 2

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

func negamax(search *searchContext, pos *Position, depth int, alpha int, beta int, ply int, allowNull bool, isCheckExtended bool, rootBestMove *Move, pvNode bool) (int, bool) {
	isRoot := rootBestMove != nil
	sideToMove := pos.SideToMove

	if searchContextStopped(search) {
		return 0, false
	}

	if !isRoot && search.history != nil && search.history[positionKey(pos)] >= 3 {
		return repetitionDrawScore, true
	}

	if !isRoot {
		search.nodes++
	}
	kingSquare := FindKing(pos)
	inCheck := InCheck(pos, kingSquare)

	if depth <= 0 {
		if !isCheckExtended && inCheck {
			return negamax(search, pos, 1, alpha, beta, ply, allowNull, true, rootBestMove, pvNode)
		}
		if !inCheck {
			return quiesce(search, pos, alpha, beta, ply)
		}
		return Eval(pos), true
	}

	key := positionKey(pos)
	originalAlpha := alpha
	var ttMove Move
	if entry, ok := search.tt.Probe(key); ok {
		ttMove = entry.Move
		if !isRoot && entry.Depth >= depth {
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
	eval := Eval(pos)
	rfpMargin := 80 * depth
	if eval > rfpMargin+beta && !pvNode && !inCheck && depth <= 6 {
		return eval, true
	}
	if !isRoot && allowNull && depth >= 3 && !inCheck && hasNonPawnMaterial(pos) && !pvNode {
		undo := MakeNullMove(pos)
		score, ok := negamax(search, pos, depth-1-nullMoveReduction, -beta, -beta+1, ply+1, false, isCheckExtended, nil, false)
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
	var scoreBuf [256]int
	moves := GeneratePseudoLegalMovesInto(pos, pseudoBuf[:0])
	var killerA, killerB Move
	if !isRoot && ply >= 0 && ply < maxSearchPly {
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

	bestMove := Move{}
	bestScore := -mateScore
	searchedMoves := 0

	for pseudoMoveIndex := range moves {
		move := pickBestMove(moves, scores, pseudoMoveIndex)
		tactical := isTacticalMove(pos, move)
		undo, legal := makeMoveIfLegal(pos, move, kingSquare)
		if !legal {
			continue
		}

		moveIndex := searchedMoves
		searchedMoves++

		repetitionKey, count := pushRepetition(search.history, pos)

		var score int
		var ok bool
		if count >= 3 {
			if isRoot {
				score = repetitionAvoidanceScore
			} else {
				score = repetitionDrawScore
			}
			ok = true
		} else {
			if !pvNode || moveIndex > 0 {
				if moveIndex >= 3 && depth >= 3 && !inCheck && !tactical {
					score, ok = negamax(search, pos, depth-1-LMReduction, -alpha-1, -alpha, ply+1, false, isCheckExtended, nil, false)
					score = -score

					if ok && score > alpha {
						score, ok = negamax(search, pos, depth-1, -alpha-1, -alpha, ply+1, true, isCheckExtended, nil, false)
						score = -score
					}
				} else {
					score, ok = negamax(search, pos, depth-1, -alpha-1, -alpha, ply+1, true, isCheckExtended, nil, false)
					score = -score
				}
			}

			if pvNode && (moveIndex == 0 || score > alpha) {
				score, ok = negamax(search, pos, depth-1, -beta, -alpha, ply+1, true, isCheckExtended, nil, true)
				score = -score
			}

		}
		popRepetition(search.history, repetitionKey)

		UnmakeMove(pos, undo)

		if !ok {
			return 0, false
		}

		if score > bestScore {
			bestScore = score
			bestMove = move
			if isRoot {
				*rootBestMove = move
			}
		}
		if score >= beta {
			if !isRoot && isQuietMove(pos, move) {
				side := 0
				if sideToMove < 0 {
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

	if searchedMoves == 0 {
		if inCheck {
			return -mateScore + ply, true
		}
		return 0, true
	}

	flag := TTExact
	if bestScore <= originalAlpha {
		flag = TTUpperBound
	}
	search.tt.Store(key, depth, bestScore, flag, bestMove)

	return bestScore, true
}

func hasNonPawnMaterial(pos *Position) bool {
	sideToMove := pos.SideToMove

	for square, piece := range pos.Board {
		if IsOffBoard(square) || piece == Empty || (piece > 0) != (sideToMove > 0) {
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
	moves := GeneratePseudoLegalMovesInto(pos, pseudoBuf[:0])
	bestMove, ok := firstLegalMove(pos, moves)
	if !ok {
		return Move{}
	}

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
			search := newSearchContext(ctx, deadline, history, tt, historyTable)
			move = bestMove
			ok := false
			score, ok = negamax(search, pos, currentDepth, alpha, beta, 0, true, false, &move, true)
			totalNodes += search.nodes

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

func firstLegalMove(pos *Position, moves []Move) (Move, bool) {
	kingSquare := FindKing(pos)

	for _, move := range moves {
		undo, ok := makeMoveIfLegal(pos, move, kingSquare)
		if !ok {
			continue
		}

		UnmakeMove(pos, undo)
		return move, true
	}

	return Move{}, false
}

func isQuietMove(pos *Position, move Move) bool {
	if move.isEnPassant || move.Promotion != 0 || move.isCastling {
		return false
	}
	return pos.Board[move.To] == Empty
}

func quiesce(search *searchContext, pos *Position, alpha int, beta int, ply int) (int, bool) {
	if searchContextStopped(search) {
		return 0, false
	}

	search.nodes++

	key := positionKey(pos)
	originalAlpha := alpha
	kingSquare := FindKing(pos)
	inCheck := InCheck(pos, kingSquare)

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
		score := Eval(pos)

		if score >= beta {
			search.tt.Store(key, 0, score, TTLowerBound, Move{})
			return score, true
		}
		if score > alpha {
			alpha = score
		}
	}

	var pseudoMoveBuffer [256]Move
	var scoreBuffer [256]int
	var moves []Move
	if inCheck {
		moves = GeneratePseudoLegalMovesInto(pos, pseudoMoveBuffer[:0])
	} else {
		moves = getPseudoTacticalMovesInto(pos, pseudoMoveBuffer[:0])
	}
	scores := scoreBuffer[:len(moves)]
	qScoreMoves(pos, moves, ttMove, scores)

	bestMove := Move{}
	searchedMoves := 0

	for pseudoMoveIndex := range moves {
		move := pickBestMove(moves, scores, pseudoMoveIndex)
		undo, legal := makeMoveIfLegal(pos, move, kingSquare)
		if !legal {
			continue
		}
		searchedMoves++

		score, ok := quiesce(search, pos, -beta, -alpha, ply+1)
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

	if inCheck && searchedMoves == 0 {
		score := -mateScore + ply
		search.tt.Store(key, 0, score, TTExact, Move{})
		return score, true
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

func getPseudoTacticalMovesInto(pos *Position, pseudoMoves []Move) []Move {
	pseudoMoves = GeneratePseudoLegalMovesInto(pos, pseudoMoves)
	return filterTacticalMoves(pos, pseudoMoves)
}

func storeKiller(killers *[maxSearchPly][2]Move, ply int, move Move) {
	if killers[ply][0] == move {
		return
	}
	killers[ply][1] = killers[ply][0]
	killers[ply][0] = move
}
