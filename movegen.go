package lacrima

import "math/bits"

const pseudoMoveCapacity = 128

var (
	pawnAttacks   [2][64]Bitboard
	knightAttacks [64]Bitboard
	kingAttacks   [64]Bitboard

	rookMagics   [64]magicEntry
	bishopMagics [64]magicEntry
	rookMoves    []Bitboard
	bishopMoves  []Bitboard
)

type magicEntry struct {
	mask   Bitboard
	magic  uint64
	shift  uint
	offset uint32
}

var rookDirections = [...]struct{ rank, file int }{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
}

var bishopDirections = [...]struct{ rank, file int }{
	{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
}

var knightDirections = [...]struct{ rank, file int }{
	{2, 1}, {2, -1}, {-2, 1}, {-2, -1},
	{1, 2}, {1, -2}, {-1, 2}, {-1, -2},
}

var kingDirections = [...]struct{ rank, file int }{
	{1, 0}, {-1, 0}, {0, 1}, {0, -1},
	{1, 1}, {1, -1}, {-1, 1}, {-1, -1},
}

func init() {
	initLeaperAttacks()
	initMagicBitboards()
}

func bit(square int) Bitboard {
	return Bitboard(1) << square
}

func onBoard(rank, file int) bool {
	return rank >= 0 && rank < 8 && file >= 0 && file < 8
}

func square(rank, file int) int {
	return rank*8 + file
}

func initLeaperAttacks() {
	for sq := 0; sq < 64; sq++ {
		rank := sq / 8
		file := sq & 7

		for _, df := range []int{-1, 1} {
			if onBoard(rank+1, file+df) {
				pawnAttacks[White][sq] |= bit(square(rank+1, file+df))
			}
			if onBoard(rank-1, file+df) {
				pawnAttacks[Black][sq] |= bit(square(rank-1, file+df))
			}
		}

		for _, dir := range knightDirections {
			toRank := rank + dir.rank
			toFile := file + dir.file
			if onBoard(toRank, toFile) {
				knightAttacks[sq] |= bit(square(toRank, toFile))
			}
		}

		for _, dir := range kingDirections {
			toRank := rank + dir.rank
			toFile := file + dir.file
			if onBoard(toRank, toFile) {
				kingAttacks[sq] |= bit(square(toRank, toFile))
			}
		}
	}
}

func initMagicBitboards() {
	for sq := 0; sq < 64; sq++ {
		entry, table := findMagic(sq, rookMask(sq), rookAttacksSlow, uint64(0x9e3779b97f4a7c15)+uint64(sq)*0xbf58476d1ce4e5b9)
		entry.offset = uint32(len(rookMoves))
		rookMagics[sq] = entry
		rookMoves = append(rookMoves, table...)

		entry, table = findMagic(sq, bishopMask(sq), bishopAttacksSlow, uint64(0x94d049bb133111eb)+uint64(sq)*0x2545f4914f6cdd1d)
		entry.offset = uint32(len(bishopMoves))
		bishopMagics[sq] = entry
		bishopMoves = append(bishopMoves, table...)
	}
}

func rookMask(sq int) Bitboard {
	rank := sq / 8
	file := sq & 7
	var mask Bitboard

	for r := rank + 1; r <= 6; r++ {
		mask |= bit(square(r, file))
	}
	for r := rank - 1; r >= 1; r-- {
		mask |= bit(square(r, file))
	}
	for f := file + 1; f <= 6; f++ {
		mask |= bit(square(rank, f))
	}
	for f := file - 1; f >= 1; f-- {
		mask |= bit(square(rank, f))
	}

	return mask
}

func bishopMask(sq int) Bitboard {
	rank := sq / 8
	file := sq & 7
	var mask Bitboard

	for r, f := rank+1, file+1; r <= 6 && f <= 6; r, f = r+1, f+1 {
		mask |= bit(square(r, f))
	}
	for r, f := rank+1, file-1; r <= 6 && f >= 1; r, f = r+1, f-1 {
		mask |= bit(square(r, f))
	}
	for r, f := rank-1, file+1; r >= 1 && f <= 6; r, f = r-1, f+1 {
		mask |= bit(square(r, f))
	}
	for r, f := rank-1, file-1; r >= 1 && f >= 1; r, f = r-1, f-1 {
		mask |= bit(square(r, f))
	}

	return mask
}

func rookAttacksSlow(sq int, blockers Bitboard) Bitboard {
	return slidingAttacksSlow(sq, blockers, rookDirections[:])
}

func bishopAttacksSlow(sq int, blockers Bitboard) Bitboard {
	return slidingAttacksSlow(sq, blockers, bishopDirections[:])
}

func slidingAttacksSlow(sq int, blockers Bitboard, directions []struct{ rank, file int }) Bitboard {
	rank := sq / 8
	file := sq & 7
	var attacks Bitboard

	for _, dir := range directions {
		for r, f := rank+dir.rank, file+dir.file; onBoard(r, f); r, f = r+dir.rank, f+dir.file {
			to := square(r, f)
			toBit := bit(to)
			attacks |= toBit
			if blockers&toBit != 0 {
				break
			}
		}
	}

	return attacks
}

func findMagic(sq int, mask Bitboard, attackFn func(int, Bitboard) Bitboard, seed uint64) (magicEntry, []Bitboard) {
	relevantBits := bits.OnesCount64(uint64(mask))
	entry := magicEntry{
		mask:  mask,
		shift: uint(64 - relevantBits),
	}
	tableSize := 1 << relevantBits
	occupancies := make([]Bitboard, 0, tableSize)
	attacks := make([]Bitboard, 0, tableSize)

	for occupancy := Bitboard(0); ; {
		occupancies = append(occupancies, occupancy)
		attacks = append(attacks, attackFn(sq, occupancy))
		occupancy = (occupancy - mask) & mask
		if occupancy == 0 {
			break
		}
	}

	used := make([]Bitboard, tableSize)
	seen := make([]uint32, tableSize)
	stamp := uint32(0)

	for {
		entry.magic = randomSparseUint64(&seed)

		stamp++
		if stamp == 0 {
			clear(seen)
			stamp = 1
		}

		collision := false
		for i, occupancy := range occupancies {
			index := magicIndex(entry, occupancy)
			if seen[index] != stamp {
				seen[index] = stamp
				used[index] = attacks[i]
				continue
			}
			if used[index] != attacks[i] {
				collision = true
				break
			}
		}

		if collision {
			continue
		}

		table := make([]Bitboard, tableSize)
		for i, occupancy := range occupancies {
			index := magicIndex(entry, occupancy)
			table[index] = attacks[i]
		}
		return entry, table
	}
}

func randomSparseUint64(seed *uint64) uint64 {
	return nextRandom(seed) & nextRandom(seed) & nextRandom(seed)
}

func nextRandom(seed *uint64) uint64 {
	*seed += 0x9e3779b97f4a7c15
	z := *seed
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

func magicIndex(entry magicEntry, blockers Bitboard) int {
	blockers &= entry.mask
	hash := uint64(blockers) * entry.magic
	return int(entry.offset) + int(hash>>entry.shift)
}

func rookAttacks(sq int, occupancy Bitboard) Bitboard {
	return rookMoves[magicIndex(rookMagics[sq], occupancy)]
}

func bishopAttacks(sq int, occupancy Bitboard) Bitboard {
	return bishopMoves[magicIndex(bishopMagics[sq], occupancy)]
}

func occupied(pos *Position) Bitboard {
	return pos.Board.Colours[White] | pos.Board.Colours[Black]
}

func appendBitboardMoves(moves []Move, from int, targets Bitboard) []Move {
	for targets != 0 {
		to := popLSB(&targets)
		moves = append(moves, Move{From: uint8(from), To: uint8(to)})
	}
	return moves
}

func popLSB(bb *Bitboard) int {
	square := bits.TrailingZeros64(uint64(*bb))
	*bb &= *bb - 1
	return square
}

func GeneratePseudoLegalMovesInto(pos *Position, moves []Move) []Move {
	if cap(moves) == 0 {
		moves = make([]Move, 0, pseudoMoveCapacity)
	}
	moves = moves[:0]

	side := pos.SideToMove
	own := pos.Board.Colours[side]
	all := occupied(pos)

	moves = generatePawnMoves(pos, moves)

	knights := pos.Board.GetPieceBoard(side, Knight)
	for knights != 0 {
		from := popLSB(&knights)
		moves = appendBitboardMoves(moves, from, knightAttacks[from]&^own)
	}

	bishops := pos.Board.GetPieceBoard(side, Bishop)
	for bishops != 0 {
		from := popLSB(&bishops)
		moves = appendBitboardMoves(moves, from, bishopAttacks(from, all)&^own)
	}

	rooks := pos.Board.GetPieceBoard(side, Rook)
	for rooks != 0 {
		from := popLSB(&rooks)
		moves = appendBitboardMoves(moves, from, rookAttacks(from, all)&^own)
	}

	queens := pos.Board.GetPieceBoard(side, Queen)
	for queens != 0 {
		from := popLSB(&queens)
		moves = appendBitboardMoves(moves, from, (rookAttacks(from, all)|bishopAttacks(from, all))&^own)
	}

	kings := pos.Board.GetPieceBoard(side, King)
	for kings != 0 {
		from := popLSB(&kings)
		moves = appendBitboardMoves(moves, from, kingAttacks[from]&^own)
		moves = generateCastlingMoves(pos, moves, from)
	}

	return moves
}

func generatePawnMoves(pos *Position, moves []Move) []Move {
	side := pos.SideToMove
	pawns := pos.Board.GetPieceBoard(side, Pawn)
	enemy := pos.Board.Colours[side^1]

	for pawns != 0 {
		from := popLSB(&pawns)
		rank := from / 8
		file := from & 7

		if side == White {
			one := from + 8
			if one < 64 && pos.PieceAt(uint8(one)) == Empty {
				if rank == 6 {
					moves = appendPromotions(moves, from, one)
				} else {
					moves = append(moves, Move{From: uint8(from), To: uint8(one)})
					if rank == 1 {
						two := from + 16
						if pos.PieceAt(uint8(two)) == Empty {
							moves = append(moves, Move{From: uint8(from), To: uint8(two), isDoublePawnPush: true})
						}
					}
				}
			}

			if file > 0 {
				moves = appendPawnCapture(pos, moves, from, from+7, enemy, rank == 6)
			}
			if file < 7 {
				moves = appendPawnCapture(pos, moves, from, from+9, enemy, rank == 6)
			}
			continue
		}

		one := from - 8
		if one >= 0 && pos.PieceAt(uint8(one)) == Empty {
			if rank == 1 {
				moves = appendPromotions(moves, from, one)
			} else {
				moves = append(moves, Move{From: uint8(from), To: uint8(one)})
				if rank == 6 {
					two := from - 16
					if pos.PieceAt(uint8(two)) == Empty {
						moves = append(moves, Move{From: uint8(from), To: uint8(two), isDoublePawnPush: true})
					}
				}
			}
		}

		if file > 0 {
			moves = appendPawnCapture(pos, moves, from, from-9, enemy, rank == 1)
		}
		if file < 7 {
			moves = appendPawnCapture(pos, moves, from, from-7, enemy, rank == 1)
		}
	}

	return moves
}

func appendPawnCapture(pos *Position, moves []Move, from int, to int, enemy Bitboard, promotes bool) []Move {
	toBit := bit(to)
	if enemy&toBit != 0 {
		if promotes {
			return appendPromotions(moves, from, to)
		}
		return append(moves, Move{From: uint8(from), To: uint8(to)})
	}

	if pos.EnPassantTarget != -1 && to == int(pos.EnPassantTarget) {
		return append(moves, Move{From: uint8(from), To: uint8(to), isEnPassant: true})
	}

	return moves
}

func appendPromotions(moves []Move, from int, to int) []Move {
	moves = append(moves,
		Move{From: uint8(from), To: uint8(to), Promotion: Knight},
		Move{From: uint8(from), To: uint8(to), Promotion: Bishop},
		Move{From: uint8(from), To: uint8(to), Promotion: Rook},
		Move{From: uint8(from), To: uint8(to), Promotion: Queen},
	)
	return moves
}

func generateCastlingMoves(pos *Position, moves []Move, from int) []Move {
	if pos.SideToMove == White {
		if from == 4 && pos.CastlingRights&WhiteKingSide != 0 &&
			pos.PieceAt(5) == Empty && pos.PieceAt(6) == Empty &&
			pos.PieceAt(7) == Rook && pos.ColourAt(7) == White {
			moves = append(moves, Move{From: 4, To: 6, isCastling: true})
		}
		if from == 4 && pos.CastlingRights&WhiteQueenSide != 0 &&
			pos.PieceAt(1) == Empty && pos.PieceAt(2) == Empty && pos.PieceAt(3) == Empty &&
			pos.PieceAt(0) == Rook && pos.ColourAt(0) == White {
			moves = append(moves, Move{From: 4, To: 2, isCastling: true})
		}
		return moves
	}

	if from == 60 && pos.CastlingRights&BlackKingSide != 0 &&
		pos.PieceAt(61) == Empty && pos.PieceAt(62) == Empty &&
		pos.PieceAt(63) == Rook && pos.ColourAt(63) == Black {
		moves = append(moves, Move{From: 60, To: 62, isCastling: true})
	}
	if from == 60 && pos.CastlingRights&BlackQueenSide != 0 &&
		pos.PieceAt(57) == Empty && pos.PieceAt(58) == Empty && pos.PieceAt(59) == Empty &&
		pos.PieceAt(56) == Rook && pos.ColourAt(56) == Black {
		moves = append(moves, Move{From: 60, To: 58, isCastling: true})
	}

	return moves
}

func IsSquareAttacked(pos *Position, sq int, byColour uint8) bool {
	if sq < 0 || sq >= 64 {
		return false
	}

	enemyPawns := pos.Board.GetPieceBoard(byColour, Pawn)
	enemyKnights := pos.Board.GetPieceBoard(byColour, Knight)
	enemyBishops := pos.Board.GetPieceBoard(byColour, Bishop)
	enemyRooks := pos.Board.GetPieceBoard(byColour, Rook)
	enemyQueens := pos.Board.GetPieceBoard(byColour, Queen)
	enemyKing := pos.Board.GetPieceBoard(byColour, King)

	all := occupied(pos)

	if pawnAttacks[byColour^1][sq]&enemyPawns != 0 {
		return true
	}
	if knightAttacks[sq]&enemyKnights != 0 {
		return true
	}
	if kingAttacks[sq]&enemyKing != 0 {
		return true
	}
	if bishopAttacks(sq, all)&(enemyBishops|enemyQueens) != 0 {
		return true
	}
	if rookAttacks(sq, all)&(enemyRooks|enemyQueens) != 0 {
		return true
	}

	return false
}

func InCheck(pos *Position, kingSquare int) bool {
	if kingSquare < 0 || kingSquare >= 64 {
		return false
	}
	return IsSquareAttacked(pos, kingSquare, pos.SideToMove^1)
}

func filterLegalMovesInto(pos *Position, moves []Move, legalMoves []Move) []Move {
	legalMoves = legalMoves[:0]
	if cap(legalMoves) < len(moves) {
		legalMoves = make([]Move, 0, len(moves))
	}

	kingSquare := currentKingSquare(pos)
	for _, move := range moves {
		undo, ok := makeMoveIfLegal(pos, move, kingSquare)
		if !ok {
			continue
		}

		legalMoves = append(legalMoves, move)
		UnmakeMove(pos, undo)
	}

	return legalMoves
}

func makeMoveIfLegal(pos *Position, move Move, kingSquare int) (Undo, bool) {
	movingSide := pos.SideToMove
	movingPiece := pos.PieceAt(move.From)

	if move.isCastling {
		opponent := movingSide ^ 1
		switch move.To {
		case 6:
			if IsSquareAttacked(pos, 4, opponent) || IsSquareAttacked(pos, 5, opponent) || IsSquareAttacked(pos, 6, opponent) {
				return Undo{}, false
			}
		case 2:
			if IsSquareAttacked(pos, 4, opponent) || IsSquareAttacked(pos, 3, opponent) || IsSquareAttacked(pos, 2, opponent) {
				return Undo{}, false
			}
		case 62:
			if IsSquareAttacked(pos, 60, opponent) || IsSquareAttacked(pos, 61, opponent) || IsSquareAttacked(pos, 62, opponent) {
				return Undo{}, false
			}
		case 58:
			if IsSquareAttacked(pos, 60, opponent) || IsSquareAttacked(pos, 59, opponent) || IsSquareAttacked(pos, 58, opponent) {
				return Undo{}, false
			}
		}
	}

	undo := MakeMove(pos, move)
	checkedKingSquare := kingSquare
	if movingPiece == King {
		checkedKingSquare = int(move.To)
	}
	if checkedKingSquare != -1 && IsSquareAttacked(pos, checkedKingSquare, pos.SideToMove) {
		UnmakeMove(pos, undo)
		return Undo{}, false
	}

	return undo, true
}

func GetLegalMoves(pos *Position) []Move {
	return getLegalMovesInto(pos, make([]Move, 0, pseudoMoveCapacity), nil)
}

func getLegalMovesInto(pos *Position, pseudoMoves []Move, legalMoves []Move) []Move {
	pseudoLegalMoves := GeneratePseudoLegalMovesInto(pos, pseudoMoves)
	return filterLegalMovesInto(pos, pseudoLegalMoves, legalMoves)
}
