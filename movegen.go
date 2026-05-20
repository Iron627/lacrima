package lacrima

func slide(pos *Position, from int) []Move {
	var moves []Move
	var directions []int
	piece := pos.Board[from]

	switch PieceType(piece) {
	case 4:
		directions = []int{N, S, E, W}
	case 3:
		directions = []int{NE, SE, NW, SW}
	case 5:
		directions = []int{N, S, E, W, NE, SE, NW, SW}
	default:
		return moves
	}

	for _, dir := range directions {
		for to := from + dir; !IsOffBoard(to); to += dir {
			target := pos.Board[to]

			if target == Empty {
				moves = append(moves, Move{From: from, To: to})
			} else {
				if !SameColor(piece, target) {
					moves = append(moves, Move{From: from, To: to})
				}
				break
			}
		}
	}
	return moves
}

func GeneratePseudoLegalMoves(pos *Position) []Move {}
