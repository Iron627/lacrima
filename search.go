package lacrima

import (
	"context"
	"math"
	stdtime "time"
)

const (
	mateScore                = 100000
	fiftyMoveRuleHalfmoves   = 100
	repetitionAvoidanceScore = -1
	repetitionDrawScore      = 0
	nullMoveReduction        = 2
	maxSearchPly             = 128
	aspirationWindow         = 50
)

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

func lmReduction(depth int, moveIndex int) int {
	moveIndex++

	return int(0.8 + math.Log(float64(depth))*math.Log(float64(moveIndex))/2.5)
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func updateHistory(history *[2][128][128]int, side int, from int, to int, bonus int) {
	if bonus > maxHistoryValue {
		bonus = maxHistoryValue
	} else if bonus < -maxHistoryValue {
		bonus = -maxHistoryValue
	}

	h := history[side][from][to]
	h += bonus - h*absInt(bonus)/maxHistoryValue
	history[side][from][to] = h
}

func negamax(search *searchContext, pos *Position, depth int, alpha int, beta int, ply int, allowNull bool, rootBestMove *Move, pvNode bool) int {
	isRoot := rootBestMove != nil
	sideToMove := pos.SideToMove

	if searchContextStopped(search) {
		search.ok = false
		return 0
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
		return quiesce(search, pos, alpha, beta, ply)
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
		score := negamax(search, pos, depth-1-nullMoveReduction, -beta, -beta+1, ply+1, false, nil, false)
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

	scores := scoreBuf[:len(moves)]
	scoreMoves(pos, moves, ttMove, killerA, killerB, search.historyTable, scores)

	bestMove := Move{}
	bestScore := -mateScore
	searchedMoves := 0
	var quietsTried [256]Move
	quietsTriedCount := 0

	extension := 0
	if inCheck {
		extension += 1
	}

	newDepth := depth + extension - 1

	for pseudoMoveIndex := range moves {
		move := pickBestMove(moves, scores, pseudoMoveIndex)
		tactical := isTacticalMove(pos, move)
		quiet := isQuietMove(pos, move)
		undo, legal := makeMoveIfLegal(pos, move, kingSquare)
		if !legal {
			continue
		}

		moveIndex := searchedMoves
		searchedMoves++

		if !isRoot && quiet && quietsTriedCount < len(quietsTried) {
			quietsTried[quietsTriedCount] = move
			quietsTriedCount++
		}

		repetitionKey, count := pushRepetition(search.history, pos)

		var score int
		if count >= 2 {
			score = drawScore(isRoot)
		} else {
			if !pvNode || moveIndex > 0 {
				if moveIndex >= 3 && depth >= 3 && !inCheck && !tactical {
					score = negamax(search, pos, newDepth-lmReduction(depth, moveIndex), -alpha-1, -alpha, ply+1, false, nil, false)
					score = -score

					if search.ok && score > alpha {
						score = negamax(search, pos, newDepth, -alpha-1, -alpha, ply+1, true, nil, false)
						score = -score
					}
				} else {
					score = negamax(search, pos, newDepth, -alpha-1, -alpha, ply+1, true, nil, false)
					score = -score
				}
			}

			if pvNode && (moveIndex == 0 || score > alpha) {
				score = negamax(search, pos, newDepth, -beta, -alpha, ply+1, true, nil, true)
				score = -score
			}
		}
		popRepetition(search.history, repetitionKey)

		UnmakeMove(pos, undo)

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
			if !isRoot && quiet {
				side := 0
				if sideToMove < 0 {
					side = 1
				}

				bonus := depth * depth
				updateHistory(search.historyTable, side, move.From, move.To, bonus)

				for i := 0; i < quietsTriedCount; i++ {
					quietMove := quietsTried[i]
					if quietMove == move {
						continue
					}

					updateHistory(search.historyTable, side, quietMove.From, quietMove.To, -bonus)
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
		deltaA := aspirationWindow
		deltaB := aspirationWindow
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
			score = negamax(search, pos, currentDepth, alpha, beta, 0, true, &move, true)
			totalNodes += search.nodes

			if !search.ok {
				return bestMove
			}

			if score <= alpha && alpha > -mateScore {
				alpha = score - deltaA
				if alpha < -mateScore {
					alpha = -mateScore
				}
				deltaA *= 2
				continue
			}

			if score >= beta && beta < mateScore {
				beta = score + deltaB
				if beta > mateScore {
					beta = mateScore
				}

				deltaB *= 2
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
