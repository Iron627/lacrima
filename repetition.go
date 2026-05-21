package lacrima

type RepetitionHistory map[uint64]int

var (
	zobristPieces    [13][128]uint64
	zobristSide      uint64
	zobristCastling  [16]uint64
	zobristEnPassant [8]uint64
)

func init() {
	var seed uint64 = 0x9e3779b97f4a7c15

	next := func() uint64 {
		seed += 0x9e3779b97f4a7c15
		z := seed
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		return z ^ (z >> 31)
	}

	for piece := range zobristPieces {
		for square := range zobristPieces[piece] {
			zobristPieces[piece][square] = next()
		}
	}

	zobristSide = next()

	for i := range zobristCastling {
		zobristCastling[i] = next()
	}

	for i := range zobristEnPassant {
		zobristEnPassant[i] = next()
	}
}

func positionKey(pos *Position) uint64 {
	var key uint64

	for square, piece := range pos.Board {
		if IsOffBoard(square) || piece == Empty {
			continue
		}

		key ^= zobristPieces[pieceZobristIndex(piece)][square]
	}

	if pos.SideToMove < 0 {
		key ^= zobristSide
	}

	key ^= zobristCastling[pos.CastlingRights&0x0f]

	if pos.EnPassantSquare != -1 {
		file := int(pos.EnPassantSquare) & 7
		key ^= zobristEnPassant[file]
	}

	return key
}

func pieceZobristIndex(piece int8) int {
	return int(piece + 6)
}

func cloneRepetitionHistory(history RepetitionHistory) RepetitionHistory {
	if history == nil {
		return nil
	}

	clone := make(RepetitionHistory, len(history))
	for key, count := range history {
		clone[key] = count
	}
	return clone
}

func pushRepetition(history RepetitionHistory, pos *Position) (uint64, int) {
	if history == nil {
		return 0, 0
	}

	key := positionKey(pos)
	history[key]++
	return key, history[key]
}

func popRepetition(history RepetitionHistory, key uint64) {
	if history == nil {
		return
	}

	history[key]--
	if history[key] == 0 {
		delete(history, key)
	}
}
