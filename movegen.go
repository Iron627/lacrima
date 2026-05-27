package lacrima

const pseudoMoveCapacity = 32

var rookDirections = [...]int{N, S, E, W}
var bishopDirections = [...]int{NE, SE, NW, SW}
var queenDirections = [...]int{N, S, E, W, NE, SE, NW, SW}
var kingDirections = [...]int{N, S, E, W, NE, SE, NW, SW}
var sideCaptureDirections = [...]int{E, W}

func slide(pos *Position, from int, moves []Move) []Move {
	piece := pos.Board[from]

	switch PieceType(piece) {
	case 4:
		for _, dir := range rookDirections {
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
	case 3:
		for _, dir := range bishopDirections {
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
	case 5:
		for _, dir := range queenDirections {
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
	default:
		return moves
	}

	return moves
}

func knight(pos *Position, from int, moves []Move) []Move {
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

func pawn(pos *Position, from int, moves []Move) []Move {
	var dir, startRank int
	piece := pos.Board[from]

	if PieceType(piece) != 1 {
		return moves
	}

	if piece > 0 {
		dir = S
		startRank = 1
	} else {
		dir = N
		startRank = 6
	}

	to := from + dir

	if !IsOffBoard(to) && pos.Board[to] == Empty {
		if (to>>4 == 7 && piece > 0) || (to>>4 == 0 && piece < 0) {
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

	for _, sideDir := range sideCaptureDirections {
		captureTo := from + dir + sideDir
		if IsOffBoard(captureTo) {
			continue
		}

		target := pos.Board[captureTo]
		if target != Empty && !SameColor(piece, target) {
			if (captureTo>>4 == 7 && piece > 0) || (captureTo>>4 == 0 && piece < 0) {
				for promo := int8(2); promo <= 5; promo++ {
					moves = append(moves, Move{From: from, To: captureTo, Promotion: promo})
				}
			} else {
				moves = append(moves, Move{From: from, To: captureTo})
			}
		}
	}

	if pos.EnPassantSquare != -1 {
		for _, sideDir := range sideCaptureDirections {
			epTo := from + dir + sideDir
			capturedSquare := from + sideDir
			if !IsOffBoard(epTo) && epTo == int(pos.EnPassantSquare) &&
				pos.Board[capturedSquare] == -piece {
				moves = append(moves, Move{From: from, To: epTo, isEnPassant: true})
			}
		}
	}

	return moves
}
func king(pos *Position, from int, moves []Move) []Move {

	piece := pos.Board[from]
	if PieceType(piece) != 6 {
		return moves
	}
	for _, dir := range kingDirections {
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
		if from == 4 && pos.CastlingRights&WhiteKingside != 0 &&
			pos.Board[5] == Empty && pos.Board[6] == Empty && pos.Board[7] == WhiteRook {
			moves = append(moves, Move{From: from, To: 6, isCastling: true})
		}
		if from == 4 && pos.CastlingRights&WhiteQueenside != 0 &&
			pos.Board[1] == Empty && pos.Board[2] == Empty && pos.Board[3] == Empty &&
			pos.Board[0] == WhiteRook {
			moves = append(moves, Move{From: from, To: 2, isCastling: true})
		}
	} else {
		if from == 116 && pos.CastlingRights&BlackKingside != 0 &&
			pos.Board[117] == Empty && pos.Board[118] == Empty && pos.Board[119] == BlackRook {
			moves = append(moves, Move{From: from, To: 118, isCastling: true})
		}
		if from == 116 && pos.CastlingRights&BlackQueenside != 0 &&
			pos.Board[113] == Empty && pos.Board[114] == Empty && pos.Board[115] == Empty &&
			pos.Board[112] == BlackRook {
			moves = append(moves, Move{From: from, To: 114, isCastling: true})
		}
	}

	return moves
}
func GeneratePseudoLegalMovesInto(pos *Position, moves []Move) []Move {
	if cap(moves) == 0 {
		moves = make([]Move, 0, pseudoMoveCapacity)
	}
	moves = moves[:0]
	for i, piece := range pos.Board {
		if IsOffBoard(i) || (piece == Empty || (piece > 0) != (pos.SideToMove > 0)) {
			continue
		}

		switch PieceType(piece) {
		case 1:
			moves = pawn(pos, i, moves)
		case 2:
			moves = knight(pos, i, moves)
		case 3, 4, 5:
			moves = slide(pos, i, moves)
		case 6:
			moves = king(pos, i, moves)
		}
	}

	return moves
}

func GeneratePseudoLegalMoves(pos *Position) []Move {
	return GeneratePseudoLegalMovesInto(pos, make([]Move, 0, pseudoMoveCapacity))
}

func filterLegalMovesInto(pos *Position, moves []Move, legalMoves []Move) []Move {
	legalMoves = legalMoves[:0]
	if cap(legalMoves) < len(moves) {
		legalMoves = make([]Move, 0, len(moves))
	}
	kingSquare := FindKing(pos, pos.SideToMove)

	for _, move := range moves {
		movingSide := pos.SideToMove
		movingPiece := pos.Board[move.From]

		if move.isCastling {
			switch move.To {
			case 6:
				if IsSquareAttacked(pos, 4, -movingSide) || IsSquareAttacked(pos, 5, -movingSide) || IsSquareAttacked(pos, 6, -movingSide) {
					continue
				}
			case 2:
				if IsSquareAttacked(pos, 4, -movingSide) || IsSquareAttacked(pos, 3, -movingSide) || IsSquareAttacked(pos, 2, -movingSide) {
					continue
				}
			case 118:
				if IsSquareAttacked(pos, 116, -movingSide) || IsSquareAttacked(pos, 117, -movingSide) || IsSquareAttacked(pos, 118, -movingSide) {
					continue
				}
			case 114:
				if IsSquareAttacked(pos, 116, -movingSide) || IsSquareAttacked(pos, 115, -movingSide) || IsSquareAttacked(pos, 114, -movingSide) {
					continue
				}
			}
		}

		undo := MakeMove(pos, move)
		checkedKingSquare := kingSquare
		if movingPiece == WhiteKing || movingPiece == BlackKing {
			checkedKingSquare = move.To
		}
		if !InCheck(pos, movingSide, checkedKingSquare) {
			legalMoves = append(legalMoves, move)
		}

		UnmakeMove(pos, undo)
	}

	return legalMoves
}

func filterLegalMoves(pos *Position, moves []Move) []Move {
	return filterLegalMovesInto(pos, moves, make([]Move, 0, len(moves)))
}

func GetLegalMoves(pos *Position, colour int8) []Move {
	return getLegalMovesInto(pos, colour, make([]Move, 0, pseudoMoveCapacity), nil)
}

func getLegalMovesInto(pos *Position, colour int8, pseudoMoves []Move, legalMoves []Move) []Move {
	originalSideToMove := pos.SideToMove
	defer func() {
		pos.SideToMove = originalSideToMove
	}()

	pos.SideToMove = colour
	pseudoLegalMoves := GeneratePseudoLegalMovesInto(pos, pseudoMoves)
	return filterLegalMovesInto(pos, pseudoLegalMoves, legalMoves)
}
