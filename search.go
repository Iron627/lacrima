package lacrima

import (
	"context"
	stdtime "time"
)

const mateScore = 100000
const fiftyMoveRuleHalfmoves = 100
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
	ok           bool
}

func newSearchContext(ctx context.Context, deadline stdtime.Time, history RepetitionHistory, tt *TranspositionTable, historyTable *[2][128][128]int) *searchContext {
	return &searchContext{
		ctx:          ctx,
		deadline:     deadline,
		history:      history,
		tt:           tt,
		historyTable: historyTable,
		ok:           true,
	}
}

func scoreToTT(score int, ply int) int {
	if score > mateScore-maxSearchPly {
		return score + ply
	}
	if score < -mateScore+maxSearchPly {
		return score - ply
	}
	return score
}

func scoreFromTT(score int, ply int) int {
	if score > mateScore-maxSearchPly {
		return score - ply
	}
	if score < -mateScore+maxSearchPly {
		return score + ply
	}
	return score
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

func isFiftyMoveRuleDraw(pos *Position) bool {
	return pos.HalfmoveClock >= fiftyMoveRuleHalfmoves
}

func drawScore(isRoot bool) int {
	if isRoot {
		return repetitionAvoidanceScore
	}
	return repetitionDrawScore
}

func negamax(search *searchContext, pos *Position, depth int, alpha int, beta int, ply int, allowNull bool, isCheckExtended bool, rootBestMove *Move, pvNode bool) int {
	isRoot := rootBestMove != nil
	sideToMove := pos.SideToMove

	if searchContextStopped(search) {
		search.ok = false
		return 0
	}

	if !isRoot && search.history != nil && search.history[positionKey(pos)] >= 3 {
		return repetitionDrawScore
	}

	if !isRoot {
		search.nodes++
	}
	kingSquare := FindKing(pos)
	inCheck := InCheck(pos, kingSquare)
	fiftyMoveDraw := isFiftyMoveRuleDraw(pos)

	if !isRoot && fiftyMoveDraw && !inCheck {
		return repetitionDrawScore
	}

	if depth <= 0 {
		if !isCheckExtended && inCheck {
			return negamax(search, pos, 1, alpha, beta, ply, allowNull, true, rootBestMove, pvNode)
		}
		if !inCheck {
			return quiesce(search, pos, alpha, beta, ply)
		}
		return Eval(pos)
	}

	key := positionKey(pos)
	originalAlpha := alpha
	var ttMove Move
	if entry, ok := search.tt.Probe(key); ok {
		ttMove = entry.Move
		ttScore := scoreFromTT(entry.Score, ply)
		if !isRoot && entry.Depth >= depth {
			switch entry.Flag {
			case TTExact:
				return ttScore
			case TTLowerBound:
				if ttScore > alpha {
					alpha = ttScore
				}
			case TTUpperBound:
				if ttScore < beta {
					beta = ttScore
				}
			}

			if alpha >= beta {
				return ttScore
			}
		}
	}
	eval := Eval(pos)
	rfpMargin := 80 * depth
	if eval > rfpMargin+beta && !pvNode && !inCheck && depth <= 6 {
		return eval
	}
	if !isRoot && allowNull && depth >= 3 && !inCheck && hasNonPawnMaterial(pos) && !pvNode {
		undo := MakeNullMove(pos)
		score := negamax(search, pos, depth-1-nullMoveReduction, -beta, -beta+1, ply+1, false, isCheckExtended, nil, false)
		score = -score
		UnmakeNullMove(pos, undo)

		if !search.ok {
			return 0
		}

		if score >= beta {
			search.tt.Store(key, depth, scoreToTT(beta, ply), TTLowerBound, Move{})
			return score
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
			return -mateScore + ply
		}
		return 0
	}

	if !isRoot && fiftyMoveDraw && inCheck {
		if _, ok := firstLegalMove(pos, moves); ok {
			return repetitionDrawScore
		}
		return -mateScore + ply
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
		var scoreKnown bool
		if score, scoreKnown = scoreAfterMoveIfFiftyMoveDraw(pos, ply, isRoot); scoreKnown {
		} else if count >= 3 {
			score = drawScore(isRoot)
			scoreKnown = true
		} else {
			if !pvNode || moveIndex > 0 {
				if moveIndex >= 3 && depth >= 3 && !inCheck && !tactical {
					score = negamax(search, pos, depth-1-LMReduction, -alpha-1, -alpha, ply+1, false, isCheckExtended, nil, false)
					score = -score

					if search.ok && score > alpha {
						score = negamax(search, pos, depth-1, -alpha-1, -alpha, ply+1, true, isCheckExtended, nil, false)
						score = -score
					}
				} else {
					score = negamax(search, pos, depth-1, -alpha-1, -alpha, ply+1, true, isCheckExtended, nil, false)
					score = -score
				}
			}

			if pvNode && (moveIndex == 0 || score > alpha) {
				score = negamax(search, pos, depth-1, -beta, -alpha, ply+1, true, isCheckExtended, nil, true)
				score = -score
			}
			scoreKnown = true
		}
		popRepetition(search.history, repetitionKey)

		UnmakeMove(pos, undo)

		if !scoreKnown {
			continue
		}

		if !search.ok {
			return 0
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

			search.tt.Store(key, depth, scoreToTT(score, ply), TTLowerBound, move)
			return score
		}

		if score > alpha {
			alpha = score
		}
	}

	if searchedMoves == 0 {
		if inCheck {
			return -mateScore + ply
		}
		return 0
	}

	flag := TTExact
	if bestScore <= originalAlpha {
		flag = TTUpperBound
	}
	search.tt.Store(key, depth, scoreToTT(bestScore, ply), flag, bestMove)

	return bestScore
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

func scoreAfterMoveIfFiftyMoveDraw(pos *Position, ply int, isRoot bool) (int, bool) {
	if !isFiftyMoveRuleDraw(pos) {
		return 0, false
	}

	kingSquare := FindKing(pos)
	if !InCheck(pos, kingSquare) {
		return drawScore(isRoot), true
	}

	var pseudoBuf [256]Move
	moves := GeneratePseudoLegalMovesInto(pos, pseudoBuf[:0])
	if _, ok := firstLegalMove(pos, moves); ok {
		return drawScore(isRoot), true
	}

	return mateScore - (ply + 1), true
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
			score = negamax(search, pos, currentDepth, alpha, beta, 0, true, false, &move, true)
			totalNodes += search.nodes

			if !search.ok {
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

func quiesce(search *searchContext, pos *Position, alpha int, beta int, ply int) int {
	if searchContextStopped(search) {
		search.ok = false
		return 0
	}

	if isFiftyMoveRuleDraw(pos) {
		return repetitionDrawScore
	}

	search.nodes++

	key := positionKey(pos)
	originalAlpha := alpha
	kingSquare := FindKing(pos)
	inCheck := InCheck(pos, kingSquare)

	var ttMove Move
	if entry, ok := search.tt.Probe(key); ok {
		ttMove = entry.Move
		ttScore := scoreFromTT(entry.Score, ply)
		switch entry.Flag {
		case TTExact:
			return ttScore
		case TTLowerBound:
			if ttScore > alpha {
				alpha = ttScore
			}
		case TTUpperBound:
			if ttScore < beta {
				beta = ttScore
			}
		}
		if alpha >= beta {
			return ttScore
		}
	}

	if !inCheck {
		score := Eval(pos)

		if score >= beta {
			search.tt.Store(key, 0, scoreToTT(score, ply), TTLowerBound, Move{})
			return score
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

		score := quiesce(search, pos, -beta, -alpha, ply+1)
		score = -score

		UnmakeMove(pos, undo)

		if !search.ok {
			return 0
		}

		if score >= beta {
			search.tt.Store(key, 0, scoreToTT(score, ply), TTLowerBound, move)
			return score
		}

		if score > alpha {
			alpha = score
			bestMove = move
		}
	}

	if inCheck && searchedMoves == 0 {
		score := -mateScore + ply
		search.tt.Store(key, 0, scoreToTT(score, ply), TTExact, Move{})
		return score
	}

	flag := TTExact
	if alpha <= originalAlpha {
		flag = TTUpperBound
	}

	search.tt.Store(key, 0, scoreToTT(alpha, ply), flag, bestMove)

	return alpha
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
