# Chess Check Detector (Go)

A minimal Go program that parses a FEN string and reports whether White or Black is in check using bitboards and Hyperbola Quintessence for sliding attacks.

## Usage

1. Ensure you have Go installed (1.20+).
2. Build and run:

```
go run .
```

3. When prompted, paste a FEN, for example (white rook on e2 checking black king on e8):

```
4k3/8/8/8/8/8/4R3/4K3 w - - 0 1
```

Expected output:

```
Board:
8  . . . . k . . .
7  . . . . . . . .
6  . . . . . . . .
5  . . . . . . . .
4  . . . . . . . .
3  . . . . . . . .
2  . . . . R . . .
1  . . . . K . . .
   a b c d e f g h
White in check? false
Black in check? true
```

After each check, you'll be prompted:
```
Check another FEN? (y/n):
```
Enter `y` to continue or `n` to quit.

You can also type `menu` at the FEN prompt to select from a predefined list of example FENs.

## Test FENs

Use these FENs to manually verify various attack scenarios.

1) Empty kings only (no checks)
- `4k3/8/8/8/8/8/8/4K3 w - - 0 1`
- Expected: White in check? false; Black in check? false

Rook checks (orthogonal)
2) White rook on e2 checking black king on e8 (open file)
- `4k3/8/8/8/8/8/4R3/4K3 w - - 0 1`
- Expected: false; true

3) Same setup but blocked by a black piece on e5 (no check)
- `4k3/8/8/8/4p3/8/4R3/4K3 w - - 0 1`
- Expected: false; false

4) Black rook on a3 checking white king on a1 (open file)
- `4k3/8/8/8/8/r7/8/K7 w - - 0 1`
- Expected: true; false

Bishop checks (diagonals)
5) White bishop on b5 checking black king on e8 (open diagonal)
- `4k3/8/8/1B6/8/8/8/4K3 w - - 0 1`
- Expected: false; true

6) Same but blocked by black piece on d7 (no check)
- `4k3/3p4/8/1B6/8/8/8/4K3 w - - 0 1`
- Expected: false; false

Queen checks (diagonal and rank/file)
7) White queen on h5 checking black king on e8 (diagonal)
- `4k3/8/8/7Q/8/8/8/4K3 w - - 0 1`
- Expected: false; true

8) White queen on e2 checking black king on e8 (file)
- `4k3/8/8/8/8/8/4Q3/4K3 w - - 0 1`
- Expected: false; true

Knight checks
9) White knight on d7 checking black king on e8
- `4k3/3N4/8/8/8/8/8/4K3 w - - 0 1`
- Expected: false; true

10) Black knight on f4 checking white king on e2
- `4k3/8/8/8/5n2/8/4K3/8 w - - 0 1`
- Expected: true; false

Pawn checks and edge masking
11) White pawn on d6 checking black king on e7 (NE)
- `8/4k3/3P4/8/8/8/8/4K3 w - - 0 1`
- Expected: false; true

12) White pawn on e6 checking black king on d7 (NW)
- `8/3k4/4P3/8/8/8/8/4K3 w - - 0 1`
- Expected: false; true

13) Black pawn on d3 checking white king on e2 (SE)
- `8/8/8/8/8/3p4/4K3/8 w - - 0 1`
- Expected: true; false

14) Black pawn on e3 checking white king on d2 (SW)
- `8/8/8/8/8/4p3/3K4/8 w - - 0 1`
- Expected: true; false

Edge-file masking validations (no wrap-around)
15) White pawn on a2 should NOT attack off-board (b3 only)
- `4k3/8/8/8/8/8/P7/4K3 w - - 0 1`

16) White pawn on h2 should NOT attack off-board (g3 only)
- `4k3/8/8/8/8/8/7P/4K3 w - - 0 1`

Mixed-piece scenarios
17) Black king on c8 checked by white rook on c1 and white bishop on b7
- `2k5/1B6/8/8/8/8/8/2R1K3 w - - 0 1`
- Expected: false; true

18) White king on e1 checked by black queen on e5 (open file)
- `4k3/8/8/4q3/8/8/8/4K3 w - - 0 1`
- Expected: true; false

19) Double check on black king: white rook on e2 and white bishop on b5 toward e8
- `4k3/8/8/1B6/8/8/4R3/4K3 w - - 0 1`
- Expected: false; true

King adjacency (rare but included)
20) Adjacent kings (both in check per adjacency rule)
- `8/8/8/8/8/8/4kK2/8 w - - 0 1`
- Expected: true; true

## Complex FENs

These more involved positions test diagonal geometry, congestion, and corrected placements.

- Diagonal check with many off-diagonal blockers: bishop a3 checks king d6
  - `8/8/3k4/8/8/B7/8/4K3 w - - 0 1`
  - Expected: White in check? false; Black in check? true

- Mixed congestion; queen check survives with a single gap on diagonal (queen d4 → h8)
  - `7k/3p4/2p5/8/3Q4/2p5/8/4K3 w - - 0 1`
  - Expected: White in check? false; Black in check? true

- Both kings surrounded; only black in check via bishop (bishop b5 → e8)
  - `4k3/8/8/1B6/8/8/8/4K3 w - - 0 1`
  - Expected: White in check? false; Black in check? true

## Notes

- Bitboard mapping: a1 = LSB (0), h8 = MSB (63).
- Pawn attack masks were corrected:
  - White: NE from NotFileH (<<9), NW from NotFileA (<<7)
  - Black: SE from NotFileH (>>7), SW from NotFileA (>>9)
- Sliding piece attacks use Hyperbola Quintessence from the king’s square and intersect with opposing sliders (rook/queen on orthogonals, bishop/queen on diagonals).