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

func knight(pos *Position, from int) []Move {
	var moves []Move
	piece := pos.Board[from]
	if PieceType(piece) != 2 {
		return moves
	}
	for _, offset := range KnightOffsets {
		to := from + offset
		if IsOffBoard(to) {
			continue
		}
		target := pos.Board[to]
		if target == Empty || !SameColor(piece, target) {
			moves = append(moves, Move{From: from, To: to})
		}
	}
	return moves
}

func pawn(pos *Position, from int) []Move {
	var moves []Move
	var dir, startRank int
	piece := pos.Board[from]

	if PieceType(piece) != 1 {
		return moves
	}

	if piece > 0 {
		dir = N
		startRank = 1
	} else {
		dir = S
		startRank = 6
	}

	to := from + dir

	if !IsOffBoard(to) && pos.Board[to] == Empty {
		if (to>>4 == 0 && piece > 0) || (to>>4 == 7 && piece < 0) {
			for promo := int8(2); promo <= 5; promo++ {
				moves = append(moves, Move{From: from, To: to, Promotion: promo})
			}
		} else {
			moves = append(moves, Move{From: from, To: to})

			if from>>4 == startRank {
				to2 := to + dir
				if !IsOffBoard(to2) && pos.Board[to2] == Empty {
					moves = append(moves, Move{From: from, To: to2, isDoublePawnPush: true})
				}
			}
		}
	}

	for _, sideDir := range []int{E, W} {
		captureTo := from + dir + sideDir
		if IsOffBoard(captureTo) {
			continue
		}

		target := pos.Board[captureTo]
		if target != Empty && !SameColor(piece, target) {
			if (captureTo>>4 == 0 && piece > 0) || (captureTo>>4 == 7 && piece < 0) {
				for promo := int8(2); promo <= 5; promo++ {
					moves = append(moves, Move{From: from, To: captureTo, Promotion: promo})
				}
			} else {
				moves = append(moves, Move{From: from, To: captureTo})
			}
		}
	}

	if pos.EnPassantSquare != -1 {
		for _, sideDir := range []int{E, W} {
			epTo := from + dir + sideDir
			if epTo == int(pos.EnPassantSquare) {
				moves = append(moves, Move{From: from, To: epTo, isEnPassant: true})
			}
		}
	}

	return moves
}
func king(pos *Position, from int) []Move {
	var moves []Move
	piece := pos.Board[from]
	if PieceType(piece) != 6 {
		return moves
	}
	for _, dir := range []int{N, S, E, W, NE, SE, NW, SW} {
		to := from + dir
		if IsOffBoard(to) {
			continue
		}
		target := pos.Board[to]
		if target == Empty || !SameColor(piece, target) {
			moves = append(moves, Move{From: from, To: to})
		}
	}
	if piece > 0 {
		if pos.CastlingRights&WhiteKingside != 0 &&
			pos.Board[5] == Empty && pos.Board[6] == Empty {
			moves = append(moves, Move{From: from, To: 6, isCastling: true})
		}
		if pos.CastlingRights&WhiteQueenside != 0 &&
			pos.Board[1] == Empty && pos.Board[2] == Empty && pos.Board[3] == Empty {
			moves = append(moves, Move{From: from, To: 2, isCastling: true})
		}
	} else {
		if pos.CastlingRights&BlackKingside != 0 &&
			pos.Board[117] == Empty && pos.Board[118] == Empty {
			moves = append(moves, Move{From: from, To: 118, isCastling: true})
		}
		if pos.CastlingRights&BlackQueenside != 0 &&
			pos.Board[113] == Empty && pos.Board[114] == Empty && pos.Board[115] == Empty {
			moves = append(moves, Move{From: from, To: 114, isCastling: true})
		}
	}
	return moves
}
func GeneratePseudoLegalMoves(pos *Position) []Move {
	var moves []Move

	for i, piece := range pos.Board {
		if piece == Empty || (piece > 0) != (pos.SideToMove > 0) {
			continue
		}

		switch PieceType(piece) {
		case 1:
			moves = append(moves, pawn(pos, i)...)
		case 2:
			moves = append(moves, knight(pos, i)...)
		case 3, 4, 5:
			moves = append(moves, slide(pos, i)...)
		case 6:
			moves = append(moves, king(pos, i)...)
		}
	}

	return moves
}
