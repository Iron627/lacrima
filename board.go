package lacrima

type Board [128]int8
type Move struct {
	From             int
	To               int
	Promotion        int8
	isCastling       bool
	isEnPassant      bool
	isDoublePawnPush bool
}

type Position struct {
	Board           Board
	SideToMove      int8
	CastlingRights  uint8
	EnPassantSquare int8
	HalfmoveClock   int
	FullmoveNumber  int
}

const (
	WhiteKingside  uint8 = 1 << 0 // 0001
	WhiteQueenside uint8 = 1 << 1 // 0010
	BlackKingside  uint8 = 1 << 2 // 0100
	BlackQueenside uint8 = 1 << 3 // 1000
)
const (
	Empty       = 0
	WhitePawn   = 1
	WhiteKnight = 2
	WhiteBishop = 3
	WhiteRook   = 4
	WhiteQueen  = 5
	BlackPawn   = -1
	BlackKnight = -2
	BlackBishop = -3
	BlackRook   = -4
	BlackQueen  = -5
	WhiteKing   = 6
	BlackKing   = -6
)

var KnightOffsets = [8]int{
	-33, -31, -18, -14,
	14, 18, 31, 33,
}

func SameColor(a, b int8) bool {
	return a != Empty && b != Empty && (a > 0) == (b > 0)
}

func PieceType(piece int8) int8 {
	if piece < 0 {
		return -piece
	}
	return piece

}
func IsOffBoard(square int) bool {
	return (square & 0x88) != 0
}
func FindKing(pos *Position, color int8) int {
	king := WhiteKing
	if color < 0 {
		king = BlackKing
	}
	for i, piece := range pos.Board {
		if int(piece) == king {
			return i
		}
	}
	return -1
}
func isSquareAttacked(pos *Position, square int, byColor int8) bool {
	if IsOffBoard(square) {
		return false
	}

	pawn := int8(WhitePawn)
	knight := int8(WhiteKnight)
	bishop := int8(WhiteBishop)
	rook := int8(WhiteRook)
	queen := int8(WhiteQueen)
	king := int8(WhiteKing)
	if byColor < 0 {
		pawn = BlackPawn
		knight = BlackKnight
		bishop = BlackBishop
		rook = BlackRook
		queen = BlackQueen
		king = BlackKing
	}

	pawnOrigins := []int{square + N + W, square + N + E}
	if byColor < 0 {
		pawnOrigins = []int{square + S + W, square + S + E}
	}
	for _, from := range pawnOrigins {
		if !IsOffBoard(from) && pos.Board[from] == pawn {
			return true
		}
	}

	for _, offset := range KnightOffsets {
		from := square + offset
		if !IsOffBoard(from) && pos.Board[from] == knight {
			return true
		}
	}

	for _, dir := range []int{N, S, E, W} {
		for from := square + dir; !IsOffBoard(from); from += dir {
			piece := pos.Board[from]
			if piece == Empty {
				continue
			}
			if piece == rook || piece == queen {
				return true
			}
			break
		}
	}

	for _, dir := range []int{NE, SE, NW, SW} {
		for from := square + dir; !IsOffBoard(from); from += dir {
			piece := pos.Board[from]
			if piece == Empty {
				continue
			}
			if piece == bishop || piece == queen {
				return true
			}
			break
		}
	}

	for _, dir := range []int{N, S, E, W, NE, SE, NW, SW} {
		from := square + dir
		if !IsOffBoard(from) && pos.Board[from] == king {
			return true
		}
	}

	return false
}

func InCheck(pos *Position, color int8) bool {
	kingSquare := FindKing(pos, color)
	if kingSquare == -1 {
		return false
	}
	return isSquareAttacked(pos, kingSquare, -color)
}

const (
	N  = -16
	S  = 16
	E  = 1
	W  = -1
	NE = -15
	NW = -17
	SE = 17
	SW = 15
)
