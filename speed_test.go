package lacrima

import (
	"context"
	"testing"
)

var (
	benchMoveSink  Move
	benchMovesSink []Move
	benchBoolSink  bool
	benchIntSink   int
	benchUintSink  uint64
	benchUndoSink  Undo
)

var benchFENs = []struct {
	name string
	fen  string
}{
	{
		name: "startpos",
		fen:  startFEN,
	},
	{
		name: "kiwipete",
		fen:  "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	},
	{
		name: "middlegame",
		fen:  "r2q1rk1/pp2bppp/2n1pn2/2bp4/3P4/2PBPN2/PP1NBPPP/R2Q1RK1 w - - 0 9",
	},
	{
		name: "endgame",
		fen:  "8/8/3k4/8/3K4/8/4P3/8 w - - 0 1",
	},
}

func BenchmarkGeneratePseudoLegalMovesInto(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		b.Run(tc.name, func(b *testing.B) {
			moves := make([]Move, 0, pseudoMoveCapacity)
			for b.Loop() {
				benchMovesSink = GeneratePseudoLegalMovesInto(&pos, moves)
			}
		})
	}
}

func BenchmarkGetLegalMoves(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				benchMovesSink = GetLegalMoves(&pos, pos.SideToMove)
			}
		})
	}
}

func BenchmarkIsSquareAttacked(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		kingSquare := FindKing(&pos, pos.SideToMove)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				benchBoolSink = IsSquareAttacked(&pos, kingSquare, -pos.SideToMove)
			}
		})
	}
}

func BenchmarkInCheck(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				kingSquare := FindKing(&pos, pos.SideToMove)
				benchBoolSink = InCheck(&pos, pos.SideToMove, kingSquare)
			}
		})
	}
}

func BenchmarkEval(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				benchIntSink = Eval(&pos, pos.SideToMove)
			}
		})
	}
}

func BenchmarkMakeUnmakeMove(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		moves := GetLegalMoves(&pos, pos.SideToMove)
		if len(moves) == 0 {
			b.Fatalf("%s has no legal moves", tc.name)
		}
		move := moves[0]

		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				undo := MakeMove(&pos, move)
				UnmakeMove(&pos, undo)
				benchUndoSink = undo
			}
		})
	}
}

func BenchmarkMoveFromUCI(b *testing.B) {
	for _, tc := range benchFENs {
		pos := mustBenchPosition(b, tc.fen)
		moves := GetLegalMoves(&pos, pos.SideToMove)
		if len(moves) == 0 {
			b.Fatalf("%s has no legal moves", tc.name)
		}
		uci := MoveToUCI(moves[len(moves)/2])

		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				var ok bool
				benchMoveSink, ok = MoveFromUCI(&pos, uci)
				benchBoolSink = ok
			}
		})
	}
}

func BenchmarkPerft(b *testing.B) {
	cases := []struct {
		name  string
		fen   string
		depth int
	}{
		{name: "startpos_d3", fen: startFEN, depth: 3},
		{name: "kiwipete_d3", fen: benchFENs[1].fen, depth: 3},
		{name: "endgame_d4", fen: benchFENs[3].fen, depth: 4},
	}

	for _, tc := range cases {
		pos := mustBenchPosition(b, tc.fen)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				benchUintSink = Perft(&pos, tc.depth)
			}
		})
	}
}

func BenchmarkSearch(b *testing.B) {
	cases := []struct {
		name  string
		fen   string
		depth int
	}{
		{name: "startpos_d3", fen: startFEN, depth: 3},
		{name: "kiwipete_d3", fen: benchFENs[1].fen, depth: 3},
		{name: "endgame_d5", fen: benchFENs[3].fen, depth: 5},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				pos := mustBenchPosition(b, tc.fen)
				benchMoveSink = searchBestMove(context.Background(), &pos, tc.depth, 0, nil, nil, NewTranspositionTable(defaultTranspositionTableEntries))
			}
		})
	}
}

func mustBenchPosition(tb testing.TB, fen string) Position {
	tb.Helper()

	pos, err := PositionFromFEN(fen)
	if err != nil {
		tb.Fatalf("parse benchmark FEN %q: %v", fen, err)
	}

	return pos
}
