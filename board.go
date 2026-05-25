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

type ScoredMove struct {
	Move  Move
	Score int
}

type Position struct {
	Board           Board
	SideToMove      int8
	CastlingRights  uint8
	EnPassantSquare int8
	HalfmoveClock   int
	FullmoveNumber  int
}
type Undo struct {
	Move            Move
	CapturedPiece   int8
	CastlingRights  uint8
	EnPassantSquare int8
	HalfmoveClock   int
	FullmoveNumber  int
}
type NullUndo struct {
	SideToMove      int8
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
func PieceValue(pieceType int8) int {
	switch pieceType {
	case 1:
		return 100
	case 2:
		return 320
	case 3:
		return 330
	case 4:
		return 500
	case 5:
		return 900
	case 6:
		return 20000
	default:
		return 0
	}
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
		if IsOffBoard(i) {
			continue
		}
		if int(piece) == king {
			return i
		}
	}
	return -1
}
func IsSquareAttacked(pos *Position, square int, byColor int8) bool {
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

func InCheck(pos *Position, color int8, kingSquare int) bool {
	if kingSquare == -1 {
		return false
	}
	return IsSquareAttacked(pos, kingSquare, -color)
}

func MakeMove(pos *Position, move Move) Undo {
	piece := pos.Board[move.From]
	capturedPiece := pos.Board[move.To]
	if move.isEnPassant {
		capturedSquare := move.To
		if piece > 0 {
			capturedSquare -= S
		} else {
			capturedSquare -= N
		}
		capturedPiece = pos.Board[capturedSquare]
	}

	undo := Undo{
		Move:            move,
		CapturedPiece:   capturedPiece,
		CastlingRights:  pos.CastlingRights,
		EnPassantSquare: pos.EnPassantSquare,
		HalfmoveClock:   pos.HalfmoveClock,
		FullmoveNumber:  pos.FullmoveNumber,
	}

	pos.Board[move.To] = piece
	pos.Board[move.From] = Empty

	if move.isDoublePawnPush {
		if piece > 0 {
			pos.EnPassantSquare = int8(move.From + S)
		} else {
			pos.EnPassantSquare = int8(move.From + N)
		}
	} else {
		pos.EnPassantSquare = -1
	}

	if move.isEnPassant {
		capturedSquare := move.To
		if piece > 0 {
			capturedSquare -= S
		} else {
			capturedSquare -= N
		}
		pos.Board[capturedSquare] = Empty
	}

	if move.Promotion != 0 {
		if piece > 0 {
			pos.Board[move.To] = move.Promotion
		} else {
			pos.Board[move.To] = -move.Promotion
		}
	}

	if move.isCastling {
		switch move.To {
		case 6:
			pos.Board[7], pos.Board[5] = Empty, WhiteRook
		case 2:
			pos.Board[0], pos.Board[3] = Empty, WhiteRook
		case 118:
			pos.Board[119], pos.Board[117] = Empty, BlackRook
		case 114:
			pos.Board[112], pos.Board[115] = Empty, BlackRook
		}
	}
	switch piece {
	case WhiteKing:
		pos.CastlingRights &^= WhiteKingside | WhiteQueenside
	case BlackKing:
		pos.CastlingRights &^= BlackKingside | BlackQueenside
	case WhiteRook:
		switch move.From {
		case 0:
			pos.CastlingRights &^= WhiteQueenside
		case 7:
			pos.CastlingRights &^= WhiteKingside
		}
	case BlackRook:
		switch move.From {
		case 112:
			pos.CastlingRights &^= BlackQueenside
		case 119:
			pos.CastlingRights &^= BlackKingside
		}
	}
	switch undo.CapturedPiece {
	case WhiteRook:
		switch move.To {
		case 0:
			pos.CastlingRights &^= WhiteQueenside
		case 7:
			pos.CastlingRights &^= WhiteKingside
		}
	case BlackRook:
		switch move.To {
		case 112:
			pos.CastlingRights &^= BlackQueenside
		case 119:
			pos.CastlingRights &^= BlackKingside
		}
	}
	if undo.CapturedPiece != Empty {
		pos.HalfmoveClock = 0
	} else if piece == WhitePawn || piece == BlackPawn {
		pos.HalfmoveClock = 0
	} else {
		pos.HalfmoveClock++
	}

	if pos.SideToMove < 0 {
		pos.FullmoveNumber++
	}

	pos.SideToMove = -pos.SideToMove
	return undo
}

func MakeNullMove(pos *Position) NullUndo {
	undo := NullUndo{
		SideToMove:      pos.SideToMove,
		EnPassantSquare: pos.EnPassantSquare,
		HalfmoveClock:   pos.HalfmoveClock,
		FullmoveNumber:  pos.FullmoveNumber,
	}

	if pos.SideToMove < 0 {
		pos.FullmoveNumber++
	}

	pos.SideToMove = -pos.SideToMove
	pos.EnPassantSquare = -1
	pos.HalfmoveClock++

	return undo
}

func UnmakeNullMove(pos *Position, undo NullUndo) {
	pos.SideToMove = undo.SideToMove
	pos.EnPassantSquare = undo.EnPassantSquare
	pos.HalfmoveClock = undo.HalfmoveClock
	pos.FullmoveNumber = undo.FullmoveNumber
}

func UnmakeMove(pos *Position, undo Undo) {
	move := undo.Move

	piece := pos.Board[move.To]
	if move.Promotion != 0 {
		if piece > 0 {
			piece = WhitePawn
		} else {
			piece = BlackPawn
		}
	}
	pos.Board[move.From] = piece
	pos.Board[move.To] = undo.CapturedPiece
	pos.EnPassantSquare = undo.EnPassantSquare
	pos.CastlingRights = undo.CastlingRights
	pos.HalfmoveClock = undo.HalfmoveClock
	pos.FullmoveNumber = undo.FullmoveNumber

	if move.isEnPassant {
		pos.Board[move.To] = Empty

		capturedSquare := move.To
		if piece > 0 {
			capturedSquare -= S
		} else {
			capturedSquare -= N
		}

		pos.Board[capturedSquare] = undo.CapturedPiece
	}

	if move.isCastling {
		switch move.To {
		case 6:
			pos.Board[7], pos.Board[5] = WhiteRook, Empty
		case 2:
			pos.Board[0], pos.Board[3] = WhiteRook, Empty
		case 118:
			pos.Board[119], pos.Board[117] = BlackRook, Empty
		case 114:
			pos.Board[112], pos.Board[115] = BlackRook, Empty
		}
	}
	pos.SideToMove = -pos.SideToMove
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
