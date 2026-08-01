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

type Move struct {
	From             uint8
	To               uint8
	Promotion        uint8
	isCastling       bool
	isEnPassant      bool
	isDoublePawnPush bool
}

type Undo struct {
	Move            Move
	Key             uint64
	CapturedPiece   uint8
	CapturedSquare  uint8
	CastlingRights  uint8
	EnPassantTarget int8
	HalfmoveClock   uint8
	FullmoveNumber  int
	WhiteKingSquare uint8
	BlackKingSquare uint8
}

type NullUndo struct {
	Key             uint64
	SideToMove      uint8
	EnPassantTarget int8
	HalfmoveClock   uint8
	FullmoveNumber  int
}

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
	Key             uint64
}

func (pos *Position) PieceAt(square uint8) uint8 {
	return pos.Mailbox[square]
}
func (pos *Position) ColourAt(square uint8) uint8 {
	mask := Bitboard(1) << square
	if pos.Board.Colours[White]&mask != 0 {
		return White
	}
	if pos.Board.Colours[Black]&mask != 0 {
		return Black
	}
	return Empty
}
func (pos *Position) addPiece(sq uint8, colour uint8, piece uint8) {
	var bb Bitboard = 1 << sq
	pos.Board.Colours[colour] |= bb
	pos.Board.Pieces[piece] |= bb
	pos.Mailbox[sq] = piece
}

func (pos *Position) removePiece(sq uint8) {
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

func (pos *Position) movePiece(from, to uint8) {
	piece := pos.PieceAt(from)
	colour := pos.ColourAt(from)
	mask := (Bitboard(1) << from) | (Bitboard(1) << to)

	pos.Board.Colours[colour] ^= mask
	pos.Board.Pieces[piece] ^= mask

	pos.Mailbox[to] = piece
	pos.Mailbox[from] = Empty
}

func isPromotion(move Move) bool {
	return move.Promotion != Pawn && move.Promotion != Empty
}

func MakeMove(pos *Position, move Move) Undo {
	piece := pos.Mailbox[move.From]
	colour := pos.ColourAt(move.From)
	captured := pos.Mailbox[move.To]
	capturedSquare := move.To

	undo := Undo{
		Move:            move,
		Key:             pos.Key,
		CapturedPiece:   captured,
		CapturedSquare:  capturedSquare,
		CastlingRights:  pos.CastlingRights,
		EnPassantTarget: pos.EnPassantTarget,
		HalfmoveClock:   pos.HalfmoveClock,
		FullmoveNumber:  pos.FullmoveNumber,
		WhiteKingSquare: pos.WhiteKingSquare,
		BlackKingSquare: pos.BlackKingSquare,
	}

	key := pos.Key
	if pos.EnPassantTarget != -1 {
		key ^= zobristEnPassant[pos.EnPassantTarget]
	}
	key ^= zobristCastling[pos.CastlingRights&0x0f]
	key ^= zobristPieces[colour][piece][move.From]

	if captured != Empty {
		key ^= zobristPieces[colour^1][captured][move.To]
	}

	pos.EnPassantTarget = -1
	if move.isDoublePawnPush {
		if colour == White {
			pos.EnPassantTarget = int8(move.From + 8)
		} else {
			pos.EnPassantTarget = int8(move.From - 8)
		}
	}

	if move.isEnPassant {
		if colour == White {
			capturedSquare = move.To - 8
		} else {
			capturedSquare = move.To + 8
		}
		undo.CapturedSquare = capturedSquare
		undo.CapturedPiece = pos.Mailbox[capturedSquare]
		key ^= zobristPieces[colour^1][undo.CapturedPiece][capturedSquare]
		pos.removePiece(capturedSquare)
	} else if captured != Empty {
		pos.removePiece(move.To)
	}

	pos.movePiece(move.From, move.To)
	key ^= zobristPieces[colour][piece][move.To]

	if isPromotion(move) {
		key ^= zobristPieces[colour][piece][move.To]
		key ^= zobristPieces[colour][move.Promotion][move.To]
		pos.Board.Pieces[piece] &^= Bitboard(1) << move.To
		pos.Board.Pieces[move.Promotion] |= Bitboard(1) << move.To
		pos.Mailbox[move.To] = move.Promotion
	}

	if piece == King {
		if colour == White {
			pos.WhiteKingSquare = move.To
		} else {
			pos.BlackKingSquare = move.To
		}
	}

	if move.isCastling {
		switch move.To {
		case 6:
			key ^= zobristPieces[colour][Rook][7] ^ zobristPieces[colour][Rook][5]
			pos.movePiece(7, 5)
		case 2:
			key ^= zobristPieces[colour][Rook][0] ^ zobristPieces[colour][Rook][3]
			pos.movePiece(0, 3)
		case 62:
			key ^= zobristPieces[colour][Rook][63] ^ zobristPieces[colour][Rook][61]
			pos.movePiece(63, 61)
		case 58:
			key ^= zobristPieces[colour][Rook][56] ^ zobristPieces[colour][Rook][59]
			pos.movePiece(56, 59)
		}
	}

	switch piece {
	case King:
		if colour == White {
			pos.CastlingRights &^= WhiteKingSide | WhiteQueenSide
		} else {
			pos.CastlingRights &^= BlackKingSide | BlackQueenSide
		}
	case Rook:
		switch move.From {
		case 0:
			pos.CastlingRights &^= WhiteQueenSide
		case 7:
			pos.CastlingRights &^= WhiteKingSide
		case 56:
			pos.CastlingRights &^= BlackQueenSide
		case 63:
			pos.CastlingRights &^= BlackKingSide
		}
	}

	switch undo.CapturedPiece {
	case Rook:
		switch undo.CapturedSquare {
		case 0:
			pos.CastlingRights &^= WhiteQueenSide
		case 7:
			pos.CastlingRights &^= WhiteKingSide
		case 56:
			pos.CastlingRights &^= BlackQueenSide
		case 63:
			pos.CastlingRights &^= BlackKingSide
		}
	}

	if piece == Pawn || undo.CapturedPiece != Empty {
		pos.HalfmoveClock = 0
	} else {
		pos.HalfmoveClock++
	}

	if pos.SideToMove == Black {
		pos.FullmoveNumber++
	}
	pos.SideToMove ^= 1
	key ^= zobristSide
	key ^= zobristCastling[pos.CastlingRights&0x0f]
	if pos.EnPassantTarget != -1 {
		key ^= zobristEnPassant[pos.EnPassantTarget]
	}
	pos.Key = key

	return undo
}

func UnmakeMove(pos *Position, undo Undo) {
	move := undo.Move
	pos.SideToMove ^= 1
	pos.CastlingRights = undo.CastlingRights
	pos.EnPassantTarget = undo.EnPassantTarget
	pos.HalfmoveClock = undo.HalfmoveClock
	pos.FullmoveNumber = undo.FullmoveNumber
	pos.WhiteKingSquare = undo.WhiteKingSquare
	pos.BlackKingSquare = undo.BlackKingSquare
	pos.Key = undo.Key

	movingPiece := pos.Mailbox[move.To]
	if isPromotion(move) {
		movingPiece = Pawn
	}
	pos.removePiece(move.To)
	pos.addPiece(move.From, pos.SideToMove, movingPiece)

	if move.isEnPassant {
		pos.addPiece(undo.CapturedSquare, pos.SideToMove^1, undo.CapturedPiece)
	} else if undo.CapturedPiece != Empty {
		pos.addPiece(move.To, pos.SideToMove^1, undo.CapturedPiece)
	}

	if move.isCastling {
		switch move.To {
		case 6:
			pos.movePiece(5, 7)
		case 2:
			pos.movePiece(3, 0)
		case 62:
			pos.movePiece(61, 63)
		case 58:
			pos.movePiece(59, 56)
		}
	}
}

func MakeNullMove(pos *Position) NullUndo {
	undo := NullUndo{
		Key:             pos.Key,
		SideToMove:      pos.SideToMove,
		EnPassantTarget: pos.EnPassantTarget,
		HalfmoveClock:   pos.HalfmoveClock,
		FullmoveNumber:  pos.FullmoveNumber,
	}
	if pos.SideToMove == Black {
		pos.FullmoveNumber++
	}
	pos.SideToMove ^= 1
	pos.EnPassantTarget = -1
	pos.HalfmoveClock++
	pos.Key ^= zobristSide
	if undo.EnPassantTarget != -1 {
		pos.Key ^= zobristEnPassant[undo.EnPassantTarget]
	}
	return undo
}

func UnmakeNullMove(pos *Position, undo NullUndo) {
	pos.Key = undo.Key
	pos.SideToMove = undo.SideToMove
	pos.EnPassantTarget = undo.EnPassantTarget
	pos.HalfmoveClock = undo.HalfmoveClock
	pos.FullmoveNumber = undo.FullmoveNumber
}
