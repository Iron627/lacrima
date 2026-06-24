package lacrima

import (
	"errors"
	"strconv"
	"strings"
)

func PositionFromFEN(fen string) (Position, error) {
	fields := strings.Fields(fen)

	if len(fields) < 4 || len(fields) > 6 {
		return Position{}, errors.New("invalid fen")
	}

	var pos Position
	pos.WhiteKingSquare = Empty
	pos.BlackKingSquare = Empty
	pos.EnPassantTarget = -1

	for i := range pos.Mailbox {
		pos.Mailbox[i] = Empty
	}

	ranks := strings.Split(fields[0], "/")

	if len(ranks) != 8 {
		return Position{}, errors.New("invalid fen")
	}

	for rank := 0; rank < 8; rank++ {
		file := 0

		for _, ch := range ranks[7-rank] {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				if file > 8 {
					return Position{}, errors.New("invalid fen")
				}
				continue
			}

			if file >= 8 {
				return Position{}, errors.New("invalid fen")
			}

			square := uint8(rank*8 + file)

			switch ch {
			case 'P':
				pos.addPiece(square, White, Pawn)
			case 'N':
				pos.addPiece(square, White, Knight)
			case 'B':
				pos.addPiece(square, White, Bishop)
			case 'R':
				pos.addPiece(square, White, Rook)
			case 'Q':
				pos.addPiece(square, White, Queen)
			case 'K':
				pos.addPiece(square, White, King)
				pos.WhiteKingSquare = square
			case 'p':
				pos.addPiece(square, Black, Pawn)
			case 'n':
				pos.addPiece(square, Black, Knight)
			case 'b':
				pos.addPiece(square, Black, Bishop)
			case 'r':
				pos.addPiece(square, Black, Rook)
			case 'q':
				pos.addPiece(square, Black, Queen)
			case 'k':
				pos.addPiece(square, Black, King)
				pos.BlackKingSquare = square
			default:
				return Position{}, errors.New("invalid fen")
			}

			file++
		}

		if file != 8 {
			return Position{}, errors.New("invalid fen")
		}
	}

	switch fields[1] {
	case "w":
		pos.SideToMove = White
	case "b":
		pos.SideToMove = Black
	default:
		return Position{}, errors.New("invalid fen")
	}

	pos.CastlingRights = 0

	if fields[2] != "-" {
		seen := map[rune]bool{}
		for _, ch := range fields[2] {
			if seen[ch] {
				return Position{}, errors.New("invalid fen")
			}
			seen[ch] = true

			switch ch {
			case 'K':
				pos.CastlingRights |= WhiteKingSide
			case 'Q':
				pos.CastlingRights |= WhiteQueenSide
			case 'k':
				pos.CastlingRights |= BlackKingSide
			case 'q':
				pos.CastlingRights |= BlackQueenSide
			default:
				return Position{}, errors.New("invalid fen")
			}
		}
	}

	if fields[3] == "-" {
		pos.EnPassantTarget = -1
	} else {
		square, ok := squareFromString(fields[3])
		if !ok {
			return Position{}, errors.New("invalid fen")
		}
		rank := square / 8
		if rank != 2 && rank != 5 {
			return Position{}, errors.New("invalid fen")
		}
		pos.EnPassantTarget = int8(square)
	}

	pos.FullmoveNumber = 1
	if len(fields) >= 5 {
		halfmoveClock, err := strconv.Atoi(fields[4])
		if err != nil || halfmoveClock < 0 || halfmoveClock > 255 {
			return Position{}, errors.New("invalid fen")
		}
		pos.HalfmoveClock = uint8(halfmoveClock)
	}

	if len(fields) >= 6 {
		fullmoveNumber, err := strconv.Atoi(fields[5])
		if err != nil || fullmoveNumber <= 0 {
			return Position{}, errors.New("invalid fen")
		}
		pos.FullmoveNumber = fullmoveNumber
	}

	return pos, nil
}
