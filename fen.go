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
	pos.WhiteKingSquare = -1
	pos.BlackKingSquare = -1

	for i := range pos.Board {
		if IsOffBoard(i) {
			continue
		}
		pos.Board[i] = Empty
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

			square := rank*16 + file

			switch ch {
			case 'P':
				pos.Board[square] = WhitePawn
			case 'N':
				pos.Board[square] = WhiteKnight
			case 'B':
				pos.Board[square] = WhiteBishop
			case 'R':
				pos.Board[square] = WhiteRook
			case 'Q':
				pos.Board[square] = WhiteQueen
			case 'K':
				pos.Board[square] = WhiteKing
				pos.WhiteKingSquare = square
			case 'p':
				pos.Board[square] = BlackPawn
			case 'n':
				pos.Board[square] = BlackKnight
			case 'b':
				pos.Board[square] = BlackBishop
			case 'r':
				pos.Board[square] = BlackRook
			case 'q':
				pos.Board[square] = BlackQueen
			case 'k':
				pos.Board[square] = BlackKing
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
		pos.SideToMove = 1
	case "b":
		pos.SideToMove = -1
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
				pos.CastlingRights |= WhiteKingside
			case 'Q':
				pos.CastlingRights |= WhiteQueenside
			case 'k':
				pos.CastlingRights |= BlackKingside
			case 'q':
				pos.CastlingRights |= BlackQueenside
			default:
				return Position{}, errors.New("invalid fen")
			}
		}
	}

	if fields[3] == "-" {
		pos.EnPassantSquare = -1
	} else {
		square, ok := squareFromString(fields[3])
		if !ok {
			return Position{}, errors.New("invalid fen")
		}
		rank := square >> 4
		if rank != 2 && rank != 5 {
			return Position{}, errors.New("invalid fen")
		}
		pos.EnPassantSquare = int8(square)
	}

	pos.FullmoveNumber = 1
	if len(fields) >= 5 {
		halfmoveClock, err := strconv.Atoi(fields[4])
		if err != nil || halfmoveClock < 0 {
			return Position{}, errors.New("invalid fen")
		}
		pos.HalfmoveClock = halfmoveClock
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
