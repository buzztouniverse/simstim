// Braille cells, as the launcher draws them: the dot at row r (0-3) and
// column c (0-1) of a cell (internal/tui/mark.go).

const BITS = [[0, 3], [1, 4], [2, 5], [6, 7]];
export const braille = {
  bit: (r, c) => 1 << BITS[r][c],
  glyph: (bits) => String.fromCharCode(0x2800 + bits),
};
