package lacrima

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
const infiniteDepth = 100
const benchDepth = 8

// fens
var benchFENs = []string{
	"r3k2r/2pb1ppp/2pp1q2/p7/1nP1B3/1P2P3/P2N1PPP/R2QK2R w KQkq - 0 14",
	"4rrk1/2p1b1p1/p1p3q1/4p3/2P2n1p/1P1NR2P/PB3PP1/3R1QK1 b - - 2 24",
	"r3qbrk/6p1/2b2pPp/p3pP1Q/PpPpP2P/3P1B2/2PB3K/R5R1 w - - 16 42",
	"6k1/1R3p2/6p1/2Bp3p/3P2q1/P7/1P2rQ1K/5R2 b - - 4 44",
	"8/8/1p2k1p1/3p3p/1p1P1P1P/1P2PK2/8/8 w - - 3 54",
	"7r/2p3k1/1p1p1qp1/1P1Bp3/p1P2r1P/P7/4R3/Q4RK1 w - - 0 36",
	"r1bq1rk1/pp2b1pp/n1pp1n2/3P1p2/2P1p3/2N1P2N/PP2BPPP/R1BQ1RK1 b - - 2 10",
	"3r3k/2r4p/1p1b3q/p4P2/P2Pp3/1B2P3/3BQ1RP/6K1 w - - 3 87",
	"2r4r/1p4k1/1Pnp4/3Qb1pq/8/4BpPp/5P2/2RR1BK1 w - - 0 42",
	"4q1bk/6b1/7p/p1p4p/PNPpP2P/KN4P1/3Q4/4R3 b - - 0 37",
	"2q3r1/1r2pk2/pp3pp1/2pP3p/P1Pb1BbP/1P4Q1/R3NPP1/4R1K1 w - - 2 34",
	"1r2r2k/1b4q1/pp5p/2pPp1p1/P3Pn2/1P1B1Q1P/2R3P1/4BR1K b - - 1 37",
	"r3kbbr/pp1n1p1P/3ppnp1/q5N1/1P1pP3/P1N1B3/2P1QP2/R3KB1R b KQkq - 0 17",
	"8/6pk/2b1Rp2/3r4/1R1B2PP/P5K1/8/2r5 b - - 16 42",
	"1r4k1/4ppb1/2n1b1qp/pB4p1/1n1BP1P1/7P/2PNQPK1/3RN3 w - - 8 29",
	"8/p2B4/PkP5/4p1pK/4Pb1p/5P2/8/8 w - - 29 68",
	"3r4/ppq1ppkp/4bnp1/2pN4/2P1P3/1P4P1/PQ3PBP/R4K2 b - - 2 20",
	"5rr1/4n2k/4q2P/P1P2n2/3B1p2/4pP2/2N1P3/1RR1K2Q w - - 1 49",
	"1r5k/2pq2p1/3p3p/p1pP4/4QP2/PP1R3P/6PK/8 w - - 1 51",
	"q5k1/5ppp/1r3bn1/1B6/P1N2P2/BQ2P1P1/5K1P/8 b - - 2 34",
	"r1b2k1r/5n2/p4q2/1ppn1Pp1/3pp1p1/NP2P3/P1PPBK2/1RQN2R1 w - - 0 22",
	"r1bqk2r/pppp1ppp/5n2/4b3/4P3/P1N5/1PP2PPP/R1BQKB1R w KQkq - 0 5",
	"r1bqr1k1/pp1p1ppp/2p5/8/3N1Q2/P2BB3/1PP2PPP/R3K2n b Q - 1 12",
	"r1bq2k1/p4r1p/1pp2pp1/3p4/1P1B3Q/P2B1N2/2P3PP/4R1K1 b - - 2 19",
	"r4qk1/6r1/1p4p1/2ppBbN1/1p5Q/P7/2P3PP/5RK1 w - - 2 25",
	"r7/6k1/1p6/2pp1p2/7Q/8/p1P2K1P/8 w - - 0 32",
	"r3k2r/ppp1pp1p/2nqb1pn/3p4/4P3/2PP4/PP1NBPPP/R2QK1NR w KQkq - 1 5",
	"3r1rk1/1pp1pn1p/p1n1q1p1/3p4/Q3P3/2P5/PP1NBPPP/4RRK1 w - - 0 12",
	"5rk1/1pp1pn1p/p3Brp1/8/1n6/5N2/PP3PPP/2R2RK1 w - - 2 20",
	"8/1p2pk1p/p1p1r1p1/3n4/8/5R2/PP3PPP/4R1K1 b - - 3 27",
	"8/4pk2/1p1r2p1/p1p4p/Pn5P/3R4/1P3PP1/4RK2 w - - 1 33",
	"8/5k2/1pnrp1p1/p1p4p/P6P/4R1PK/1P3P2/4R3 b - - 1 38",
	"8/8/1p1kp1p1/p1pr1n1p/P6P/1R4P1/1P3PK1/1R6 b - - 15 45",
	"8/8/1p1k2p1/p1prp2p/P2n3P/6P1/1P1R1PK1/4R3 b - - 5 49",
	"8/8/1p4p1/p1p2k1p/P2npP1P/4K1P1/1P6/3R4 w - - 6 54",
	"8/8/1p4p1/p1p2k1p/P2n1P1P/4K1P1/1P6/6R1 b - - 6 59",
	"8/5k2/1p4p1/p1pK3p/P2n1P1P/6P1/1P6/4R3 b - - 14 63",
	"8/1R6/1p1K1kp1/p6p/P1p2P1P/6P1/1Pn5/8 w - - 0 67",
	"1rb1rn1k/p3q1bp/2p3p1/2p1p3/2P1P2N/PP1RQNP1/1B3P2/4R1K1 b - - 4 23",
	"4rrk1/pp1n1pp1/q5p1/P1pP4/2n3P1/7P/1P3PB1/R1BQ1RK1 w - - 3 22",
	"r2qr1k1/pb1nbppp/1pn1p3/2ppP3/3P4/2PB1NN1/PP3PPP/R1BQR1K1 w - - 4 12",
	"2r2k2/8/4P1R1/1p6/8/P4K1N/7b/2B5 b - - 0 55",
	"6k1/5pp1/8/2bKP2P/2P5/p4PNb/B7/8 b - - 1 44",
	"2rqr1k1/1p3p1p/p2p2p1/P1nPb3/2B1P3/5P2/1PQ2NPP/R1R4K w - - 3 25",
	"r1b2rk1/p1q1ppbp/6p1/2Q5/8/4BP2/PPP3PP/2KR1B1R b - - 2 14",
	"6r1/5k2/p1b1r2p/1pB1p1p1/1Pp3PP/2P1R1K1/2P2P2/3R4 w - - 1 36",
	"rnbqkb1r/pppppppp/5n2/8/2PP4/8/PP2PPPP/RNBQKBNR b KQkq - 0 2",
	"2rr2k1/1p4bp/p1q1p1p1/4Pp1n/2PB4/1PN3P1/P3Q2P/2RR2K1 w - f6 0 20",
	"3br1k1/p1pn3p/1p3n2/5pNq/2P1p3/1PN3PP/P2Q1PB1/4R1K1 w - - 0 23",
	"2r2b2/5p2/5k2/p1r1pP2/P2pB3/1P3P2/K1P3R1/7R w - - 23 93",
}

func RunUCI() {
	if len(os.Args) > 1 && os.Args[1] == "bench" {
		RunBench(os.Stdout)
		return
	}

	RunUCIWithIO(os.Stdin, os.Stdout, os.Stderr)
}

func RunBench(output io.Writer) {
	tt := NewTranspositionTable(transpositionTableEntriesForMB(defaultHashMB))
	var nodes uint64

	start := time.Now()
	for _, fen := range benchFENs {
		pos, err := PositionFromFEN(fen)
		if err != nil {
			continue
		}

		var historyTable [2][128][128]int
		var searchedNodes uint64
		tt.Clear()
		searchBestMove(context.Background(), &pos, benchDepth, 0, nil, func(info SearchInfo) {
			searchedNodes = info.Nodes
		}, tt, &historyTable)
		nodes += searchedNodes
	}
	elapsed := time.Since(start)

	nps := uint64(0)
	if elapsed > 0 {
		nps = uint64(float64(nodes) / elapsed.Seconds())
	}

	fmt.Fprintf(output, "%d nodes %d nps\n", nodes, nps)
}

func RunUCIWithIO(input io.Reader, output io.Writer, errOutput io.Writer) {
	scanner := bufio.NewScanner(input)

	pos, _ := PositionFromFEN(startFEN)
	history := RepetitionHistory{positionKey(&pos): 1}
	hashMB := defaultHashMB
	tt := NewTranspositionTable(transpositionTableEntriesForMB(hashMB))
	var historyTable [2][128][128]int

	var searchID atomic.Uint64
	var searchCancel context.CancelFunc
	var searchDone <-chan struct{}
	var outputMu sync.Mutex

	writeLine := func(args ...any) {
		outputMu.Lock()
		defer outputMu.Unlock()
		fmt.Fprintln(output, args...)
	}

	stopSearch := func(wait bool) {
		if searchCancel != nil {
			searchCancel()
		}
		if wait && searchDone != nil {
			<-searchDone
		}
		searchCancel = nil
		searchDone = nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)

		switch fields[0] {

		case "uci":
			writeLine("id name Lacrima v1.1.1")
			writeLine("id author Iron")
			writeLine("option name Hash type spin default", defaultHashMB, "min", minHashMB, "max", maxHashMB)
			writeLine("option name Threads type spin default 1 min 1 max 1")
			writeLine("option name Clear Hash type button")
			writeLine("uciok")

		case "isready":
			writeLine("readyok")

		case "setoption":
			switch {
			case isClearHashOption(fields):
				stopSearch(true)
				tt.Clear()
			case func() bool {
				nextHashMB, ok := parseHashOption(fields)
				if !ok {
					return false
				}
				stopSearch(true)
				hashMB = nextHashMB
				tt = NewTranspositionTable(transpositionTableEntriesForMB(hashMB))
				return true
			}():
			}

		case "ucinewgame":
			stopSearch(true)
			pos, _ = PositionFromFEN(startFEN)
			history = RepetitionHistory{positionKey(&pos): 1}
			tt.Clear()
			historyTable = [2][128][128]int{}

		case "quit":
			stopSearch(true)
			return

		case "stop":
			stopSearch(true)

		case "position":
			p, h, ok := parsePositionWithHistory(fields)
			if ok {
				pos = p
				history = h
			}

		case "go":
			stopSearch(true)

			id := searchID.Add(1)
			ctx, cancel := context.WithCancel(context.Background())
			searchCancel = cancel
			done := make(chan struct{})
			searchDone = done

			searchPos := pos
			searchHistory := cloneRepetitionHistory(history)

			depth, moveTime := parseGo(fields, searchPos.SideToMove)

			go func() {
				defer close(done)

				best := searchBestMove(ctx, &searchPos, depth, moveTime, searchHistory, func(info SearchInfo) {
					if searchID.Load() != id {
						return
					}

					writeLine(formatUCIInfo(info))
				}, tt, &historyTable)

				if searchID.Load() != id {
					return
				}

				writeLine("bestmove", MoveToUCI(best))
			}()

		case "perft":
			if len(fields) < 2 {
				continue
			}

			depth, err := strconv.Atoi(fields[1])
			if err != nil {
				continue
			}

			writeLine(Perft(&pos, depth))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(errOutput, err)
	}
}

func formatUCIInfo(info SearchInfo) string {
	return strings.Join([]string{
		"info",
		"depth", strconv.Itoa(info.Depth),
		formatUCIScore(info.Score),
		"nodes", strconv.FormatUint(info.Nodes, 10),
		"nps", strconv.FormatUint(info.NPS(), 10),
		"time", strconv.FormatInt(info.TimeMillis, 10),
		"pv", formatUCIPV(info),
	}, " ")
}

func formatUCIScore(score int) string {
	if score > mateScore-maxSearchPly {
		return "score mate " + strconv.Itoa((mateScore-score+1)/2)
	}
	if score < -mateScore+maxSearchPly {
		return "score mate " + strconv.Itoa(-(mateScore+score+1)/2)
	}
	return "score cp " + strconv.Itoa(score)
}

func formatUCIPV(info SearchInfo) string {
	pv := info.PV
	if len(pv) == 0 && info.BestMove != (Move{}) {
		pv = []Move{info.BestMove}
	}

	moves := make([]string, 0, len(pv))
	for _, move := range pv {
		moves = append(moves, MoveToUCI(move))
	}
	return strings.Join(moves, " ")
}

func parseGo(fields []string, stm uint8) (int, int) {
	depth := 5

	var wtime, btime, winc, binc int
	var moveTime int
	infinite := false
	hasExplicitDepth := false

	for i := 1; i < len(fields); i++ {
		switch fields[i] {

		case "depth":
			if i+1 < len(fields) {
				depth, _ = strconv.Atoi(fields[i+1])
				hasExplicitDepth = true
				i++
			}

		case "infinite":
			infinite = true
			depth = infiniteDepth
			moveTime = 0

		case "movetime":
			if i+1 < len(fields) {
				moveTime, _ = strconv.Atoi(fields[i+1])
				if !hasExplicitDepth {
					depth = infiniteDepth
				}
				i++
			}

		case "wtime":
			if i+1 < len(fields) {
				wtime, _ = strconv.Atoi(fields[i+1])
				i++
			}

		case "btime":
			if i+1 < len(fields) {
				btime, _ = strconv.Atoi(fields[i+1])
				i++
			}

		case "winc":
			if i+1 < len(fields) {
				winc, _ = strconv.Atoi(fields[i+1])
				i++
			}

		case "binc":
			if i+1 < len(fields) {
				binc, _ = strconv.Atoi(fields[i+1])
				i++
			}
		}
	}

	if moveTime == 0 && !infinite {
		timeLeft := btime
		inc := binc

		if stm == White {
			timeLeft = wtime
			inc = winc
		}

		if timeLeft > 0 {
			if !hasExplicitDepth {
				depth = infiniteDepth
			}

			moveTime = timeLeft/30 + inc/2

			if moveTime < 50 {
				moveTime = 50
			}
		}
	}

	return depth, moveTime
}

func parseHashOption(fields []string) (int, bool) {
	name, value, ok := parseSetOption(fields)
	if !ok || !strings.EqualFold(name, "Hash") || value == "" {
		return 0, false
	}

	hashMB, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	if hashMB < minHashMB {
		hashMB = minHashMB
	}
	if hashMB > maxHashMB {
		hashMB = maxHashMB
	}

	return hashMB, true
}

func isClearHashOption(fields []string) bool {
	name, _, ok := parseSetOption(fields)
	return ok && strings.EqualFold(name, "Clear Hash")
}

func parseSetOption(fields []string) (string, string, bool) {
	if len(fields) < 5 || fields[0] != "setoption" {
		return "", "", false
	}

	nameStart := -1
	valueIndex := -1
	for i := 1; i < len(fields); i++ {
		switch fields[i] {
		case "name":
			nameStart = i + 1
		case "value":
			valueIndex = i
		}
	}
	if nameStart == -1 {
		return "", "", false
	}

	if valueIndex == -1 {
		return strings.Join(fields[nameStart:], " "), "", true
	}
	if nameStart >= valueIndex {
		return "", "", false
	}
	if valueIndex+1 >= len(fields) {
		return strings.Join(fields[nameStart:valueIndex], " "), "", true
	}
	return strings.Join(fields[nameStart:valueIndex], " "), fields[valueIndex+1], true
}

func parsePositionWithHistory(fields []string) (Position, RepetitionHistory, bool) {
	if len(fields) < 2 {
		return Position{}, nil, false
	}

	var pos Position
	var err error

	i := 1

	switch fields[i] {

	case "startpos":
		pos, err = PositionFromFEN(startFEN)
		if err != nil {
			return Position{}, nil, false
		}
		i++

	case "fen":
		i++

		start := i

		for i < len(fields) && fields[i] != "moves" {
			i++
		}

		fen := strings.Join(fields[start:i], " ")

		pos, err = PositionFromFEN(fen)
		if err != nil {
			return Position{}, nil, false
		}

	default:
		return Position{}, nil, false
	}

	history := RepetitionHistory{positionKey(&pos): 1}

	if i < len(fields) {
		if fields[i] != "moves" {
			return Position{}, nil, false
		}
		i++
	}

	for ; i < len(fields); i++ {
		move, ok := MoveFromUCI(&pos, fields[i])
		if !ok {
			return Position{}, nil, false
		}
		MakeMove(&pos, move)
		history[positionKey(&pos)]++
	}

	return pos, history, true
}

func Perft(pos *Position, depth int) uint64 {
	if depth == 0 {
		return 1
	}

	moves := GetLegalMoves(pos)

	if depth == 1 {
		return uint64(len(moves))
	}

	var nodes uint64

	for _, move := range moves {
		undo := MakeMove(pos, move)

		nodes += Perft(pos, depth-1)

		UnmakeMove(pos, undo)
	}

	return nodes
}

func MoveToUCI(move Move) string {
	if move == (Move{}) {
		return "0000"
	}

	s := squareToString(move.From) + squareToString(move.To)

	if isPromotion(move) {
		switch move.Promotion {
		case Knight:
			s += "n"
		case Bishop:
			s += "b"
		case Rook:
			s += "r"
		case Queen:
			s += "q"
		}
	}

	return s
}

func MoveFromUCI(pos *Position, s string) (Move, bool) {
	moves := GetLegalMoves(pos)

	for _, move := range moves {
		if MoveToUCI(move) == s {
			return move, true
		}
	}

	return Move{}, false
}

func squareToString(square uint8) string {
	file := square & 7
	rank := square / 8

	return string(rune('a'+file)) + string(rune('1'+rank))
}

func squareFromString(s string) (uint8, bool) {
	if len(s) != 2 {
		return 0, false
	}
	if s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return 0, false
	}

	file := uint8(s[0] - 'a')
	rank := uint8(s[1] - '1')

	return rank*8 + file, true
}
