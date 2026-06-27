package lacrima

func StaticExchangeEvaluation(pos *Position, move Move) int {
	from := int(move.From)
	to := int(move.To)

	movingPiece := pos.PieceAt(move.From)
	if movingPiece == Empty {
		return 0
	}

	capturedPiece := pos.PieceAt(move.To)

	fromBB := bit(from)
	toBB := bit(to)

	occ := occupied(pos)

	if move.isEnPassant {
		capturedPiece = Pawn

		var capturedPawnSq int
		if pos.SideToMove == White {
			capturedPawnSq = to - 8
		} else {
			capturedPawnSq = to + 8
		}

		occ ^= fromBB
		occ ^= bit(capturedPawnSq)
		occ ^= toBB
	} else if capturedPiece == Empty {
		occ ^= fromBB
		occ ^= toBB
	} else {
		occ ^= fromBB
	}

	pieceOnTarget := movingPiece
	if move.Promotion != Empty {
		pieceOnTarget = move.Promotion
	}

	var gain [32]int
	depth := 0

	gain[0] = seeValue(capturedPiece)

	side := pos.SideToMove ^ 1
	attackers := seeAttackersTo(pos, to, occ)

	for {
		var attackerPiece uint8
		attackerBB := seeLeastValuableAttacker(pos, attackers, side, &attackerPiece)
		if attackerBB == 0 {
			break
		}

		depth++
		gain[depth] = seeValue(pieceOnTarget) - gain[depth-1]

		occ ^= attackerBB
		pieceOnTarget = attackerPiece
		side ^= 1

		attackers = seeAttackersTo(pos, to, occ)
	}

	for depth > 0 {
		gain[depth-1] = -max(-gain[depth-1], gain[depth])
		depth--
	}

	return gain[0]
}

func seeAttackersTo(pos *Position, sq int, occ Bitboard) Bitboard {
	pawns := (pawnAttacks[Black][sq] & pos.Board.GetPieceBoard(White, Pawn)) |
		(pawnAttacks[White][sq] & pos.Board.GetPieceBoard(Black, Pawn))

	knights := knightAttacks[sq] &
		(pos.Board.GetPieceBoard(White, Knight) |
			pos.Board.GetPieceBoard(Black, Knight))

	bishops := bishopAttacks(sq, occ) &
		(pos.Board.GetPieceBoard(White, Bishop) |
			pos.Board.GetPieceBoard(Black, Bishop) |
			pos.Board.GetPieceBoard(White, Queen) |
			pos.Board.GetPieceBoard(Black, Queen))

	rooks := rookAttacks(sq, occ) &
		(pos.Board.GetPieceBoard(White, Rook) |
			pos.Board.GetPieceBoard(Black, Rook) |
			pos.Board.GetPieceBoard(White, Queen) |
			pos.Board.GetPieceBoard(Black, Queen))

	kings := kingAttacks[sq] &
		(pos.Board.GetPieceBoard(White, King) |
			pos.Board.GetPieceBoard(Black, King))

	return (pawns | knights | bishops | rooks | kings) & occ
}

func seeLeastValuableAttacker(pos *Position, attackers Bitboard, colour uint8, piece *uint8) Bitboard {
	if bb := attackers & pos.Board.GetPieceBoard(colour, Pawn); bb != 0 {
		*piece = Pawn
		return bb & -bb
	}
	if bb := attackers & pos.Board.GetPieceBoard(colour, Knight); bb != 0 {
		*piece = Knight
		return bb & -bb
	}
	if bb := attackers & pos.Board.GetPieceBoard(colour, Bishop); bb != 0 {
		*piece = Bishop
		return bb & -bb
	}
	if bb := attackers & pos.Board.GetPieceBoard(colour, Rook); bb != 0 {
		*piece = Rook
		return bb & -bb
	}
	if bb := attackers & pos.Board.GetPieceBoard(colour, Queen); bb != 0 {
		*piece = Queen
		return bb & -bb
	}
	if bb := attackers & pos.Board.GetPieceBoard(colour, King); bb != 0 {
		*piece = King
		return bb & -bb
	}

	*piece = Empty
	return 0
}

func seeValue(piece uint8) int {
	switch piece {
	case Pawn:
		return 100
	case Knight:
		return 320
	case Bishop:
		return 330
	case Rook:
		return 500
	case Queen:
		return 900
	case King:
		return 20000
	default:
		return 0
	}
}
