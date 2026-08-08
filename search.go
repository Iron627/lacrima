package lacrima

import (
	"context"
	"math"
	stdtime "time"
)

const (
	nullMoveReduction = 2
	aspirationWindow  = 50
	fpMarginOffset    = 250
	fpMarginMult      = 60
	lmpOffset         = 5
	lmpMult           = 2
)
const (
	maxSearchPly             = 128
	mateScore                = 100000
	fiftyMoveRuleHalfmoves   = 100
	repetitionAvoidanceScore = -1
	repetitionDrawScore      = 0
	mated                    = -mateScore + maxSearchPly
)

type SearchInfo struct {
	Depth      int
	Score      int
	Nodes      uint64
	TimeMillis int64
	BestMove   Move
	PV         []Move
}

func (info SearchInfo) NPS() uint64 {
	if info.Nodes == 0 {
		return 0
	}
	if info.TimeMillis <= 0 {
		return info.Nodes * 1000
	}
	return info.Nodes * 1000 / uint64(info.TimeMillis)
}

type SearchInfoFunc func(SearchInfo)

type searchContext struct {
	ctx          context.Context
	deadline     stdtime.Time
	history      RepetitionHistory
	tt           *TranspositionTable
	historyTable *[2][128][128]int
	contHist     *[2][6][64][6][64]int
	moves        moveStack
	killers      [maxSearchPly][2]Move
	pvTable      [maxSearchPly][maxSearchPly]Move
	pvLength     [maxSearchPly]int
	nodes        uint64
	ok           bool
}

func newSearchContext(ctx context.Context, deadline stdtime.Time, history RepetitionHistory, tt *TranspositionTable, historyTable *[2][128][128]int, contHist *[2][6][64][6][64]int) *searchContext {
	return &searchContext{
		ctx:          ctx,
		deadline:     deadline,
		history:      history,
		tt:           tt,
		historyTable: historyTable,
		contHist:     contHist,
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

func currentKingSquare(pos *Position) int {
	square := pos.WhiteKingSquare
	if pos.SideToMove == Black {
		square = pos.BlackKingSquare
	}
	if square == Empty {
		return -1
	}
	return int(square)
}

func drawScore(isRoot bool) int {
	if isRoot {
		return repetitionAvoidanceScore
	}
	return repetitionDrawScore
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func updateHistory(history *[2][128][128]int, side int, from uint8, to uint8, bonus int) {
	if bonus > maxHistoryValue {
		bonus = maxHistoryValue
	} else if bonus < -maxHistoryValue {
		bonus = -maxHistoryValue
	}

	h := history[side][from][to]
	h += bonus - h*absInt(bonus)/maxHistoryValue
	history[side][from][to] = h
}

func updateContHistory(contHist *[2][6][64][6][64]int, side int, prevPiece uint8, prevTo uint8, piece uint8, to uint8, bonus int) {
	if bonus > maxHistoryValue {
		bonus = maxHistoryValue
	} else if bonus < -maxHistoryValue {
		bonus = -maxHistoryValue
	}
	h := contHist[side][prevPiece][prevTo][piece][to]
	h += bonus - h*absInt(bonus)/maxHistoryValue
	contHist[side][prevPiece][prevTo][piece][to] = h
}

func negamax(search *searchContext, pos *Position, depth int, alpha int, beta int, ply int, allowNull bool, rootBestMove *Move, pvNode bool) int {
	isRoot := rootBestMove != nil
	sideToMove := pos.SideToMove

	if pvNode && ply >= 0 && ply < maxSearchPly {
		search.pvLength[ply] = ply
	}

	if searchContextStopped(search) {
		search.ok = false
		return 0
	}

	if !isRoot {
		search.nodes++
	}
	kingSquare := currentKingSquare(pos)
	inCheck := InCheck(pos, kingSquare)
	fiftyMoveDraw := isFiftyMoveRuleDraw(pos)

	if !isRoot && fiftyMoveDraw && !inCheck {
		return repetitionDrawScore
	}

	if depth <= 0 {
		return quiesce(search, pos, alpha, beta, ply)
	}

	key := pos.Key
	originalAlpha := alpha
	var ttMove Move
	if entry, ok := search.tt.Probe(key); ok {
		ttMove = entry.Move
		ttScore := scoreFromTT(entry.Score, ply)
		if !isRoot && !pvNode && entry.Depth >= depth {
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
	previousMove, hasPreviousMove := search.moves.previous(0)
	scoreMoves(pos, moves, ttMove, killerA, killerB, search.historyTable, search.contHist, previousMove, hasPreviousMove, scores)

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
		if eval+fpMarginOffset+fpMarginMult*depth <= alpha && !inCheck && !tactical && bestScore > mated && depth <= 5 && !isRoot {
			continue
		}
		if searchedMoves >= lmpOffset+depth*depth*lmpMult && !inCheck && !tactical && bestScore > mated && !isRoot {
			break
		}
		undo, legal := makeMoveIfLegal(pos, move, kingSquare)
		if !legal {
			continue
		}
		search.moves.push(move)

		moveIndex := searchedMoves
		searchedMoves++

		if pvNode && ply+1 < maxSearchPly {
			search.pvLength[ply+1] = ply + 1
		}

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
					score = negamax(search, pos, newDepth-int(LMRTable[depth][moveIndex]), -alpha-1, -alpha, ply+1, false, nil, false)
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
		search.moves.pop()

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
				side := int(sideToMove)

				bonus := depth * depth
				updateHistory(search.historyTable, side, move.From, move.To, bonus)

				for i := 0; i < quietsTriedCount; i++ {
					quietMove := quietsTried[i]
					if quietMove == move {
						continue
					}

					updateHistory(search.historyTable, side, quietMove.From, quietMove.To, -bonus)
				}
				prevMove, ok := search.moves.previous(0)
				if ok {
					updateContHistory(search.contHist, side, pos.PieceAt(prevMove.To), prevMove.To, pos.PieceAt(move.From), move.To, bonus)
				}
				if ply >= 0 && ply < maxSearchPly {
					storeKiller(&search.killers, ply, move)
				}
			}

			search.tt.Store(key, depth, scoreToTT(score, ply), TTLowerBound, move)
			return score
		}

		if score > alpha {
			if pvNode {
				updatePV(search, ply, move)
			}
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

func updatePV(search *searchContext, ply int, move Move) {
	if ply < 0 || ply >= maxSearchPly {
		return
	}

	search.pvTable[ply][ply] = move
	nextLength := ply + 1
	if ply+1 < maxSearchPly && search.pvLength[ply+1] > nextLength {
		nextLength = search.pvLength[ply+1]
	}

	for nextPly := ply + 1; nextPly < nextLength; nextPly++ {
		search.pvTable[ply][nextPly] = search.pvTable[ply+1][nextPly]
	}
	search.pvLength[ply] = nextLength
}

func hasNonPawnMaterial(pos *Position) bool {
	return pos.Board.Colours[pos.SideToMove]&^(pos.Board.Pieces[Pawn]|pos.Board.Pieces[King]) != 0
}

func searchBestMove(ctx context.Context, pos *Position, depth int, time int, history RepetitionHistory, onInfo SearchInfoFunc, tt *TranspositionTable, historyTable *[2][128][128]int, contHist *[2][6][64][6][64]int, increment int) Move {
	originalSideToMove := pos.SideToMove
	startTime := stdtime.Now()

	defer func() {
		pos.SideToMove = originalSideToMove
	}()

	var deadline stdtime.Time
	if time > 0 {
		deadline = stdtime.Now().Add(stdtime.Duration(clampMax(int64(time/2+int(float64(increment)/1.1)), int64(time))) * stdtime.Millisecond)
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

	var hardbound int64
	var baseSoftbound int64
	if time > 0 {
		hardbound = clampMax(
			int64(time/2)+int64(float64(increment)/1.1),
			int64(time),
		)
		baseSoftbound = clampMax(
			int64(time*3/100)+int64(float64(increment)/1.1),
			hardbound,
		)
	}

	lastBestMoveChange := 0

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
		var pv []Move

		for {
			search := newSearchContext(ctx, deadline, history, tt, historyTable, contHist)
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

			pv = searchPV(search)
			break
		}

		if move != bestMove {
			lastBestMoveChange = currentDepth
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
				PV:         pv,
			})
		}

		if time > 0 {
			elapsed := stdtime.Since(startTime).Milliseconds()

			iterationsStable := currentDepth - lastBestMoveChange
			factor := math.Max(
				0.8,
				1.2-0.05*float64(iterationsStable),
			)

			softbound := clampMax(
				int64(float64(baseSoftbound)*factor),
				hardbound,
			)

			if elapsed >= softbound {
				break
			}

			if elapsed >= hardbound {
				break
			}
		}

		if searchStopped(ctx, deadline) {
			break
		}
	}

	return bestMove
}
func searchPV(search *searchContext) []Move {
	if search == nil || search.pvLength[0] <= 0 {
		return nil
	}

	length := search.pvLength[0]
	if length > maxSearchPly {
		length = maxSearchPly
	}

	pv := make([]Move, length)
	copy(pv, search.pvTable[0][:length])
	return pv
}

func firstLegalMove(pos *Position, moves []Move) (Move, bool) {
	kingSquare := currentKingSquare(pos)

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
	if move.isEnPassant || isPromotion(move) || move.isCastling {
		return false
	}
	return pos.PieceAt(move.To) == Empty
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

	key := pos.Key
	originalAlpha := alpha
	kingSquare := currentKingSquare(pos)
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
	bestScore := alpha
	searchedMoves := 0

	for pseudoMoveIndex := range moves {
		move := pickBestMove(moves, scores, pseudoMoveIndex)

		undo, legal := makeMoveIfLegal(pos, move, kingSquare)
		if !legal {
			continue
		}
		search.moves.push(move)
		searchedMoves++

		score := quiesce(search, pos, -beta, -alpha, ply+1)
		score = -score
		if score > bestScore {
			bestScore = score
		}

		search.moves.pop()
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
		if bestScore > mated && searchedMoves > 2 {
			break
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
	if move.isEnPassant || isPromotion(move) {
		return true
	}
	return pos.PieceAt(move.To) != Empty
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
func clampMin(a int64, min int64) int64 {
	if a < min {
		return min
	}
	return a
}
func clampMax(a int64, max int64) int64 {
	if a > max {
		return max
	}
	return a
}
