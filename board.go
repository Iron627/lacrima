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
