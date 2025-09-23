package main

import (
	"bufio"
	"fmt"
	"math/bits"
	"os"
	"strings"
)

type Color int

const (
	White Color = iota
	Black
)

const (
	A1 = 0
	H1 = 7
	A8 = 56
	H8 = 63
)

const (
	FileA uint64 = 0x0101010101010101
	FileH uint64 = FileA << 7
	NotFileA     = ^FileA
	NotFileH     = ^FileH
)

// Position holds bitboards per piece and color.
// All bitboards use Little-Endian Rank-File mapping: a1=LSB (0), h8=MSB (63).
type Position struct {
	whitePawns, whiteKnights, whiteBishops, whiteRooks, whiteQueens, whiteKing uint64
	blackPawns, blackKnights, blackBishops, blackRooks, blackQueens, blackKing uint64
}

func (p Position) White() uint64 {
	return p.whitePawns | p.whiteKnights | p.whiteBishops | p.whiteRooks | p.whiteQueens | p.whiteKing
}
func (p Position) Black() uint64 {
	return p.blackPawns | p.blackKnights | p.blackBishops | p.blackRooks | p.blackQueens | p.blackKing
}
func (p Position) All() uint64 { return p.White() | p.Black() }

func (p Position) KingSquare(c Color) (int, bool) {
	var bb uint64
	if c == White {
		bb = p.whiteKing
	} else {
		bb = p.blackKing
	}
	if bb == 0 {
		return 0, false
	}
	return bits.TrailingZeros64(bb), true
}

// ---------------------- FEN parsing ------------------------

func parseFEN(fen string) (Position, error) {
	// Accept full FEN but only piece placement is needed for check.
	parts := strings.Fields(fen)
	if len(parts) == 0 {
		return Position{}, fmt.Errorf("empty FEN")
	}

	boardPart := parts[0]
	ranks := strings.Split(boardPart, "/")
	if len(ranks) != 8 {
		return Position{}, fmt.Errorf("invalid FEN: expected 8 ranks, got %d", len(ranks))
	}

	var pos Position

	// FEN ranks go 8 -> 1. Our mapping is row 0 = rank1, row 7 = rank8.
	for r := 0; r < 8; r++ {
		fenRank := ranks[r]
		boardRank := 7 - r // map: first FEN rank is 8 -> index 7
		file := 0
		for _, ch := range fenRank {
			switch {
			case ch >= '1' && ch <= '8':
				file += int(ch - '0')
			default:
				if file >= 8 {
					return Position{}, fmt.Errorf("too many squares on rank %d", 8-r)
				}
				sq := boardRank*8 + file
				setPieceAt(&pos, ch, sq)
				file++
			}
		}
		if file != 8 {
			return Position{}, fmt.Errorf("incomplete rank at rank %d", 8-r)
		}
	}
	return pos, nil
}

func setPieceAt(p *Position, ch rune, sq int) {
	b := uint64(1) << uint(sq)
	switch ch {
	case 'P':
		p.whitePawns |= b
	case 'N':
		p.whiteKnights |= b
	case 'B':
		p.whiteBishops |= b
	case 'R':
		p.whiteRooks |= b
	case 'Q':
		p.whiteQueens |= b
	case 'K':
		p.whiteKing |= b
	case 'p':
		p.blackPawns |= b
	case 'n':
		p.blackKnights |= b
	case 'b':
		p.blackBishops |= b
	case 'r':
		p.blackRooks |= b
	case 'q':
		p.blackQueens |= b
	case 'k':
		p.blackKing |= b
	default:
		// ignore unknown (shouldn't happen for valid FEN)
	}
}

// ---------------------- Masks and attacks ------------------------

// Rank mask: all bits on the same rank as sq (including sq).
func rankMask(sq int) uint64 {
	r := sq / 8
	return 0xFF << (8 * r)
}

// File mask: all bits on the same file as sq (including sq).
func fileMask(sq int) uint64 {
	f := sq % 8
	return FileA << uint(f)
}

// Diagonal mask (a1-h8 slope), including sq.
func diagMask(sq int) uint64 {
	r := sq / 8
	f := sq % 8
	var m uint64
	// include sq
	m |= 1 << uint(sq)
	// NE
	for rr, ff := r+1, f+1; rr < 8 && ff < 8; rr, ff = rr+1, ff+1 {
		m |= 1 << uint(rr*8+ff)
	}
	// SW
	for rr, ff := r-1, f-1; rr >= 0 && ff >= 0; rr, ff = rr-1, ff-1 {
		m |= 1 << uint(rr*8+ff)
	}
	return m
}

// Anti-diagonal mask (h1-a8 slope), including sq.
func antiDiagMask(sq int) uint64 {
	r := sq / 8
	f := sq % 8
	var m uint64
	// include sq
	m |= 1 << uint(sq)
	// NW
	for rr, ff := r+1, f-1; rr < 8 && ff >= 0; rr, ff = rr+1, ff-1 {
		m |= 1 << uint(rr*8+ff)
	}
	// SE
	for rr, ff := r-1, f+1; rr >= 0 && ff < 8; rr, ff = rr-1, ff+1 {
		m |= 1 << uint(rr*8+ff)
	}
	return m
}

// Hyperbola Quintessence generalized form.
// See chessprogramming.org Hyperbola_Quintessence (Java section).
// Given:
// - occ: occupancy bitboard (all pieces)
// - mask: mask of the line (rank/file/diag/antidiag) including sq
// - sq: the origin square for ray generation
// Returns sliding attacks from sq along that line (includes the first blocker, excludes sq).
func lineAttacksHQ(occ uint64, mask uint64, sq int) uint64 {
	// Use the classic Hyperbola Quintessence form:
	// attacks = ((occ & mask) - 2*s) ^ reverse(reverse(occ & mask) - 2*reverse(s))
	// where s is the bit at sq.
	r := occ & mask
	s := uint64(1) << uint(sq)
	revR := bits.Reverse64(r)
	revS := bits.Reverse64(s)

	att := (r - (s << 1)) ^ bits.Reverse64(revR-(revS<<1))
	return att & mask
}

func bishopAttacksHQ(sq int, occ uint64) uint64 {
	return lineAttacksHQ(occ, diagMask(sq), sq) | lineAttacksHQ(occ, antiDiagMask(sq), sq)
}
func rookAttacksHQ(sq int, occ uint64) uint64 {
	return lineAttacksHQ(occ, rankMask(sq), sq) | lineAttacksHQ(occ, fileMask(sq), sq)
}
func queenAttacksHQ(sq int, occ uint64) uint64 {
	return bishopAttacksHQ(sq, occ) | rookAttacksHQ(sq, occ)
}

// Precomputed non-sliding attacks (knight, king) and pawn attacks from squares.
var knightAttacks [64]uint64
var kingAttacks [64]uint64
var pawnAttacksWhiteFrom [64]uint64
var pawnAttacksBlackFrom [64]uint64

func onBoard(r, f int) bool { return r >= 0 && r < 8 && f >= 0 && f < 8 }

func init() {
	// Precompute knight moves
	for sq := 0; sq < 64; sq++ {
		r := sq / 8
		f := sq % 8
		deltas := [8][2]int{
			{+2, +1}, {+2, -1}, {-2, +1}, {-2, -1},
			{+1, +2}, {+1, -2}, {-1, +2}, {-1, -2},
		}
		var m uint64
		for _, d := range deltas {
			rr, ff := r+d[0], f+d[1]
			if onBoard(rr, ff) {
				m |= 1 << uint(rr*8+ff)
			}
		}
		knightAttacks[sq] = m
	}

	// Precompute king moves
	for sq := 0; sq < 64; sq++ {
		r := sq / 8
		f := sq % 8
		var m uint64
		for dr := -1; dr <= 1; dr++ {
			for df := -1; df <= 1; df++ {
				if dr == 0 && df == 0 {
					continue
				}
				rr, ff := r+dr, f+df
				if onBoard(rr, ff) {
					m |= 1 << uint(rr*8+ff)
				}
			}
		}
		kingAttacks[sq] = m
	}

	// Precompute pawn attacks from squares
	for sq := 0; sq < 64; sq++ {
		r := sq / 8
		f := sq % 8

		// White pawns move "up" (to higher ranks): attacks to NE (r+1, f+1), NW (r+1, f-1)
		var wm uint64
		if onBoard(r+1, f+1) {
			wm |= 1 << uint((r+1)*8+(f+1))
		}
		if onBoard(r+1, f-1) {
			wm |= 1 << uint((r+1)*8+(f-1))
		}
		pawnAttacksWhiteFrom[sq] = wm

		// Black pawns move "down": attacks to SE (r-1, f+1), SW (r-1, f-1)
		var bm uint64
		if onBoard(r-1, f+1) {
			bm |= 1 << uint((r-1)*8+(f+1))
		}
		if onBoard(r-1, f-1) {
			bm |= 1 << uint((r-1)*8+(f-1))
		}
		pawnAttacksBlackFrom[sq] = bm
	}
}

// ---------------------- Check detection ------------------------

// IsSquareAttacked returns true if square sq is attacked by color c in pos.
func IsSquareAttacked(pos *Position, sq int, c Color) bool {
	occ := pos.All()

	// Pawns: check attackers to sq by c's pawns
	if c == White {
		// Correct source-file masking for white pawn attacks:
		// NE (+9) cannot originate from file H; NW (+7) cannot originate from file A.
		if (pawnAttacksWhite(pos.whitePawns) & (1 << uint(sq))) != 0 {
			return true
		}
	} else {
		// For black pawn attacks:
		// SE (-7) cannot originate from file H; SW (-9) cannot originate from file A.
		if (pawnAttacksBlack(pos.blackPawns) & (1 << uint(sq))) != 0 {
			return true
		}
	}

	// Knights
	if (knightAttacks[sq] & knightBB(pos, c)) != 0 {
		return true
	}

	// Kings (edge-case/illegal FEN but keep for completeness)
	if (kingAttacks[sq] & kingBB(pos, c)) != 0 {
		return true
	}

	// Sliding pieces
	// Generate rays from the target square and see if the first blocker is a slider of color c.
	bbBishQueens := bishopBB(pos, c) | queenBB(pos, c)
	if (bishopAttacksHQ(sq, occ) & bbBishQueens) != 0 {
		return true
	}
	bbRookQueens := rookBB(pos, c) | queenBB(pos, c)
	if (rookAttacksHQ(sq, occ) & bbRookQueens) != 0 {
		return true
	}
	return false
}

func knightBB(p *Position, c Color) uint64 {
	if c == White {
		return p.whiteKnights
	}
	return p.blackKnights
}
func bishopBB(p *Position, c Color) uint64 {
	if c == White {
		return p.whiteBishops
	}
	return p.blackBishops
}
func rookBB(p *Position, c Color) uint64 {
	if c == White {
		return p.whiteRooks
	}
	return p.blackRooks
}
func queenBB(p *Position, c Color) uint64 {
	if c == White {
		return p.whiteQueens
	}
	return p.blackQueens
}
func kingBB(p *Position, c Color) uint64 {
	if c == White {
		return p.whiteKing
	}
	return p.blackKing
}

// Pawn attack generators from bitboards (for check tests)
func pawnAttacksWhite(pawns uint64) uint64 {
	// Correct masking: mask the source files before shifting.
	// White NE is +9, invalid from source on file H -> mask NotFileH.
	// White NW is +7, invalid from source on file A -> mask NotFileA.
	attacksNE := (pawns & NotFileH) << 9
	attacksNW := (pawns & NotFileA) << 7
	return attacksNE | attacksNW
}
func pawnAttacksBlack(pawns uint64) uint64 {
	// Black SE is -7 (>>7), invalid from source on file H -> mask NotFileH.
	// Black SW is -9 (>>9), invalid from source on file A -> mask NotFileA.
	attacksSE := (pawns & NotFileH) >> 7
	attacksSW := (pawns & NotFileA) >> 9
	return attacksSE | attacksSW
}

// ---------------------- Utilities ------------------------

func printBoard(pos Position) {
	// Build a mailbox-like representation to print
	board := make([]rune, 64)
	for i := 0; i < 64; i++ {
		board[i] = '.'
	}
	set := func(bb uint64, ch rune) {
		for bb != 0 {
			s := bits.TrailingZeros64(bb)
			board[s] = ch
			bb &= bb - 1
		}
	}
	set(pos.whitePawns, 'P')
	set(pos.whiteKnights, 'N')
	set(pos.whiteBishops, 'B')
	set(pos.whiteRooks, 'R')
	set(pos.whiteQueens, 'Q')
	set(pos.whiteKing, 'K')
	set(pos.blackPawns, 'p')
	set(pos.blackKnights, 'n')
	set(pos.blackBishops, 'b')
	set(pos.blackRooks, 'r')
	set(pos.blackQueens, 'q')
	set(pos.blackKing, 'k')

	fmt.Println("Board:")
	for r := 7; r >= 0; r-- {
		fmt.Printf("%d  ", r+1)
		for f := 0; f < 8; f++ {
			fmt.Printf("%c ", board[r*8+f])
		}
		fmt.Println()
	}
	fmt.Println("   a b c d e f g h")
}

func predefinedFENs() []struct {
	name string
	fen  string
} {
	return []struct {
		name string
		fen  string
	}{
		{"Empty kings only (no checks)", "4k3/8/8/8/8/8/8/4K3 w - - 0 1"},
		{"Rook e2 checks black king e8", "4k3/8/8/8/8/8/4R3/4K3 w - - 0 1"},
		{"Rook e2 blocked by e5 (no check)", "4k3/8/8/8/4p3/8/4R3/4K3 w - - 0 1"},
		{"Black rook a3 checks white king a1", "4k3/8/8/8/8/r7/8/K7 w - - 0 1"},
		{"Bishop b5 checks black king e8", "4k3/8/8/1B6/8/8/8/4K3 w - - 0 1"},
		{"Bishop b5 blocked by d7 (no check)", "4k3/3p4/8/1B6/8/8/8/4K3 w - - 0 1"},
		{"Queen h5 checks black king e8", "4k3/8/8/7Q/8/8/8/4K3 w - - 0 1"},
		{"Queen e2 checks black king e8", "4k3/8/8/8/8/8/4Q3/4K3 w - - 0 1"},
		{"Knight d7 checks black king e8", "4k3/3N4/8/8/8/8/8/4K3 w - - 0 1"},
		{"Black knight f4 checks white king e2", "4k3/8/8/8/5n2/8/4K3/8 w - - 0 1"},
		{"White pawn d6 checks black king e7 (NE)", "8/4k3/3P4/8/8/8/8/4K3 w - - 0 1"},
		{"White pawn e6 checks black king d7 (NW)", "8/3k4/4P3/8/8/8/8/4K3 w - - 0 1"},
		{"Black pawn d3 checks white king e2 (SE)", "8/8/8/8/8/3p4/4K3/8 w - - 0 1"},
		{"Black pawn e3 checks white king d2 (SW)", "8/8/8/8/8/4p3/3K4/8 w - - 0 1"},
		{"White pawn a2 edge (b3 only)", "4k3/8/8/8/8/8/P7/4K3 w - - 0 1"},
		{"White pawn h2 edge (g3 only)", "4k3/8/8/8/8/8/7P/4K3 w - - 0 1"},
		{"Rook c1 and Bishop b7 check black king c8", "2k5/1B6/8/8/8/8/8/2R1K3 w - - 0 1"},
		{"Black queen e5 checks white king e1", "4k3/8/8/4q3/8/8/8/4K3 w - - 0 1"},
		{"Double check: bishop b5 and rook e2 on e8", "4k3/8/8/1B6/8/8/4R3/4K3 w - - 0 1"},
		{"Adjacent kings both in check", "8/8/8/8/8/8/4kK2/8 w - - 0 1"},
		{"Complex: bishop a3 checks king d6", "8/8/3k4/8/8/B7/8/4K3 w - - 0 1"},
		{"Complex: queen d4 to h8 diagonal check", "7k/3p4/2p5/8/3Q4/2p5/8/4K3 w - - 0 1"},
		{"Complex: bishop b5 checks e8", "4k3/8/8/1B6/8/8/8/4K3 w - - 0 1"},
	}
}

func selectPredefinedFEN(in *bufio.Reader) (string, bool) {
	items := predefinedFENs()
	fmt.Println("Predefined FENs:")
	for i, it := range items {
		fmt.Printf("  %2d) %s\n", i+1, it.name)
	}
	fmt.Println("  0) Cancel")
	fmt.Print("Select a number: ")
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	if line == "0" {
		return "", false
	}
	// simple parse to integer
	n := 0
	for _, ch := range line {
		if ch < '0' || ch > '9' {
			n = -1
			break
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 || n > len(items) {
		fmt.Println("Invalid selection.")
		return "", false
	}
	return items[n-1].fen, true
}

func main() {
	in := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter FEN (or type 'menu' to choose a predefined):")
		fen, _ := in.ReadString('\n')
		fen = strings.TrimSpace(fen)
		if strings.EqualFold(fen, "menu") {
			if sel, ok := selectPredefinedFEN(in); ok {
				fen = sel
			} else {
				// back to main prompt
				fmt.Println("No selection made.")
				goto CONTINUE_PROMPT
			}
		}
		if fen == "" {
			fmt.Println("No FEN provided.")
		} else {
			pos, err := parseFEN(fen)
			if err != nil {
				fmt.Println("FEN parse error:", err)
			} else {
				printBoard(pos)

				// Determine check status for both sides
				whiteKingSq, okW := pos.KingSquare(White)
				blackKingSq, okB := pos.KingSquare(Black)
				if !okW || !okB {
					fmt.Printf("Note: Missing king(s): whitePresent=%v, blackPresent=%v\n", okW, okB)
				}

				whiteInCheck := false
				blackInCheck := false
				if okW {
					whiteInCheck = IsSquareAttacked(&pos, whiteKingSq, Black)
				}
				if okB {
					blackInCheck = IsSquareAttacked(&pos, blackKingSq, White)
				}

				fmt.Printf("White in check? %v\n", whiteInCheck)
				fmt.Printf("Black in check? %v\n", blackInCheck)
			}
		}

	CONTINUE_PROMPT:
		// Prompt to continue or quit
		fmt.Print("Check another FEN? (y/n): ")
		resp, _ := in.ReadString('\n')
		resp = strings.TrimSpace(resp)
		if len(resp) == 0 || strings.ToLower(resp)[0] != 'y' {
			break
		}
	}
}