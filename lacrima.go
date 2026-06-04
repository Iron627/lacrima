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
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
const infiniteDepth = 100

func RunUCI() {
	RunUCIWithIO(os.Stdin, os.Stdout, os.Stderr)
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
			writeLine("id name Lacrima v1.0.6")
			writeLine("id author Iron")
			writeLine("option name Hash type spin default", defaultHashMB, "min", minHashMB, "max", maxHashMB)
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

					writeLine(
						"info depth", info.Depth,
						"score cp", info.Score,
						"nodes", info.Nodes,
						"time", info.TimeMillis,
						"pv", MoveToUCI(info.BestMove),
					)
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

func parseGo(fields []string, stm int8) (int, int) {
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

		if stm > 0 {
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

	moves := GetLegalMoves(pos, pos.SideToMove)

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

	if move.Promotion != 0 {
		switch move.Promotion {
		case WhiteKnight:
			s += "n"
		case WhiteBishop:
			s += "b"
		case WhiteRook:
			s += "r"
		case WhiteQueen:
			s += "q"
		}
	}

	return s
}

func MoveFromUCI(pos *Position, s string) (Move, bool) {
	moves := GetLegalMoves(pos, pos.SideToMove)

	for _, move := range moves {
		if MoveToUCI(move) == s {
			return move, true
		}
	}

	return Move{}, false
}

func squareToString(square int) string {
	file := square & 7
	rank := square >> 4

	return string(rune('a'+file)) + string(rune('1'+rank))
}

func squareFromString(s string) (int, bool) {
	if len(s) != 2 {
		return 0, false
	}
	if s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return 0, false
	}

	file := int(s[0] - 'a')
	rank := int(s[1] - '1')

	return rank*16 + file, true
}
