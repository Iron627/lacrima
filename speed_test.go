package lacrima

import (
	"context"
	"testing"
)

var (
	benchMoveSink Move
	benchUintSink uint64
)

var benchPositions = []struct {
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

type benchSearchCase struct {
	name  string
	fen   string
	depth int
}

func searchDepthCases() []benchSearchCase {
	cases := []benchSearchCase{
		{name: "startpos_d8", fen: startFEN, depth: 8},
		{name: "kiwipete_d7", fen: benchPositions[1].fen, depth: 7},
		{name: "middlegame_d7", fen: benchPositions[2].fen, depth: 7},
		{name: "endgame_d8", fen: benchPositions[3].fen, depth: 8},
	}

	return cases
}

func BenchmarkSearchDepth(b *testing.B) {
	cases := searchDepthCases()

	for _, tc := range cases {
		base := mustBenchPosition(b, tc.fen)
		tt := NewTranspositionTable(transpositionTableEntriesForMB(defaultHashMB))

		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			var totalNodes uint64
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				pos := base
				tt.Clear()
				var historyTable [2][128][128]int
				var contHist [2][6][64][6][64]int
				var searchedNodes uint64
				b.StartTimer()

				benchMoveSink = searchBestMove(context.Background(), &pos, tc.depth, 0, nil, func(info SearchInfo) {
					searchedNodes = info.Nodes
				}, tt, &historyTable, &contHist, 0)
				totalNodes += searchedNodes
			}

			if elapsed := b.Elapsed(); elapsed > 0 {
				b.ReportMetric(float64(totalNodes)/elapsed.Seconds(), "nodes/s")
			}
		})
	}
}

func BenchmarkPerftBaseline(b *testing.B) {
	cases := []struct {
		name  string
		fen   string
		depth int
	}{
		{name: "startpos_d4", fen: startFEN, depth: 4},
		{name: "kiwipete_d3", fen: benchPositions[1].fen, depth: 3},
		{name: "middlegame_d3", fen: benchPositions[2].fen, depth: 3},
		{name: "endgame_d5", fen: benchPositions[3].fen, depth: 5},
	}

	for _, tc := range cases {
		base := mustBenchPosition(b, tc.fen)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			var totalNodes uint64
			for i := 0; i < b.N; i++ {
				pos := base
				nodes := Perft(&pos, tc.depth)
				benchUintSink = nodes
				totalNodes += nodes
			}

			if elapsed := b.Elapsed(); elapsed > 0 {
				b.ReportMetric(float64(totalNodes)/elapsed.Seconds(), "nodes/s")
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
