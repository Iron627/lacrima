package lacrima

const Empty uint8 = 255
const (
	Pawn = iota
	Knight
	Bishop
	Rook
	Queen
	King
)
const (
	White = 0
	Black = 1
)
const (
	WhiteKingSide  uint8 = 1 << 0
	WhiteQueenSide uint8 = 1 << 1
	BlackKingSide  uint8 = 1 << 2
	BlackQueenSide uint8 = 1 << 3
)

type Mailbox [64]uint8
type Bitboard uint64

type Board struct {
	Colours [2]Bitboard
	Pieces  [6]Bitboard
}

func (board *Board) GetPieceBoard(colour uint8, piece uint8) Bitboard {
	return board.Pieces[piece] & board.Colours[colour]
}

type Position struct {
	Board           Board
	Mailbox         Mailbox
	SideToMove      uint8
	CastlingRights  uint8
	HalfmoveClock   uint8
	FullmoveNumber  int
	EnPassantTarget int8
	WhiteKingSquare uint8
	BlackKingSquare uint8
}

func (pos *Position) PieceAt(square uint8) uint8 {
	return pos.Mailbox[square]
}
func (pos *Position) ColourAt(square uint8) uint8 {
	mask := Bitboard(1) << square
	if pos.Board.Colours[White]&mask != 0 {
		return White
	}
	return Black
}
func (pos *Position) AddPiece(sq uint8, colour uint8, piece uint8) {
	var bb Bitboard = 1 << sq
	pos.Board.Colours[colour] |= bb
	pos.Board.Pieces[piece] |= bb
	pos.Mailbox[sq] = piece
}

func (pos *Position) RemovePiece(sq uint8) {
	var bb Bitboard = 1 << sq
	piece := pos.PieceAt(sq)
	if piece == Empty {
		return
	}
	colour := pos.ColourAt(sq)
	pos.Board.Colours[colour] &^= bb
	pos.Board.Pieces[piece] &^= bb
	pos.Mailbox[sq] = Empty
}

func (pos *Position) MovePiece(from, to uint8) {
	piece := pos.PieceAt(from)
	colour := pos.ColourAt(from)
	mask := (Bitboard(1) << from) | (Bitboard(1) << to)

	pos.Board.Colours[colour] ^= mask
	pos.Board.Pieces[piece] ^= mask

	pos.Mailbox[to] = piece
	pos.Mailbox[from] = Empty
}
