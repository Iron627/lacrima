package lacrima

type RepetitionHistory map[uint64]int

var (
	zobristPieces    [2][6][64]uint64
	zobristSide      uint64
	zobristCastling  [16]uint64
	zobristEnPassant [64]uint64
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

	for colour := range zobristPieces {
		for piece := range zobristPieces[colour] {
			for square := range zobristPieces[colour][piece] {
				zobristPieces[colour][piece][square] = next()
			}
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

	for colour := 0; colour < 2; colour++ {
		for piece := 0; piece < 6; piece++ {
			pieces := pos.Board.GetPieceBoard(uint8(colour), uint8(piece))
			for pieces != 0 {
				square := popLSB(&pieces)
				key ^= zobristPieces[colour][piece][square]
			}
		}
	}

	if pos.SideToMove == Black {
		key ^= zobristSide
	}

	key ^= zobristCastling[pos.CastlingRights&0x0f]

	if pos.EnPassantTarget != -1 {
		key ^= zobristEnPassant[pos.EnPassantTarget]
	}

	return key
}

func initPositionKey(pos *Position) {
	pos.Key = positionKey(pos)
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

	key := pos.Key
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
