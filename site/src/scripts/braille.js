// Braille drawing with no browser in it, so the build can draw each
// figure's still frame into the HTML and the page can animate from there.

// The dot at row r (0-3) and column c (0-1) of a cell, as the launcher
// draws it (internal/tui/mark.go).
const BITS = [[0, 3], [1, 4], [2, 5], [6, 7]];
export const braille = {
  bit: (r, c) => 1 << BITS[r][c],
  glyph: (bits) => String.fromCharCode(0x2800 + bits),
};

export const NOISE = "⡏⢸⣿⠿⡿⢿⣻⣽⣾⣷⣯⣟⡷⢾⣼⣧";

// The welcome mark: three strands, as rows of [glyph bits, dot count].
const STRANDS = [
  [1.0, 1.0, 1.2, 0],
  [0.8, 1.6, -0.9, 2.2],
  [0.55, 2.4, 1.7, 4.4],
];

export function markLines(t, grow, w = 44, h = 7) {
  const dw = w * 2;
  const dh = h * 4;
  const dots = Array.from({ length: dh }, () => new Uint8Array(dw));
  const mid = (dh - 1) / 2;
  for (const [amp, wave, speed, phase] of STRANDS) {
    let prev = -1;
    for (let x = 0; x < dw; x++) {
      const u = (x + 0.5) / dw;
      if (Math.abs(u - 0.5) > grow / 2) {
        prev = -1;
        continue;
      }
      const env = Math.pow(Math.sin(Math.PI * u), 1.2) * grow;
      const v = mid + amp * env * (dh / 2 - 1) * Math.sin(2 * Math.PI * u * wave + speed * t + phase);
      let y = v >= 0 ? Math.floor(v + 0.5) : -Math.floor(-v + 0.5);
      y = Math.min(Math.max(y, 0), dh - 1);
      dots[y][x] = 1;
      if (prev >= 0) for (let yy = Math.min(prev, y) + 1; yy < Math.max(prev, y); yy++) dots[yy][x] = 1;
      prev = y;
    }
  }
  const out = [];
  for (let cy = 0; cy < h; cy++) {
    const row = [];
    for (let cx = 0; cx < w; cx++) {
      let r = 0;
      let n = 0;
      for (let dy = 0; dy < 4; dy++) {
        for (let dx = 0; dx < 2; dx++) {
          if (dots[cy * 4 + dy][cx * 2 + dx]) {
            r |= braille.bit(dy, dx);
            n++;
          }
        }
      }
      row.push([r, n]);
    }
    out.push(row);
  }
  return out;
}


// The mark as plain text: its still frame.
export function markText(t = 0, grow = 1) {
  return markLines(t, grow)
    .map((row) => row.map(([r]) => (r ? braille.glyph(r) : " ")).join(""))
    .join("\n");
}

// A trace of output over time, one cell per step: two bars a cell, each 1
// to 4 dots high, made from a few slow waves so it reads like real bursts.
export function traceCells(n, seed = 0) {
  const level = (i) => {
    const v = 0.5 + 0.35 * Math.sin(i * 0.37 + seed) + 0.25 * Math.sin(i * 1.13 + 1 + seed) + 0.15 * Math.sin(i * 2.9);
    return Math.max(1, Math.min(4, Math.round(v * 4)));
  };
  const out = [];
  for (let i = 0; i < n; i++) {
    let bits = 0;
    const a = level(2 * i);
    const b = level(2 * i + 1);
    for (let r = 0; r < 4; r++) {
      if (4 - r <= a) bits |= braille.bit(r, 0);
      if (4 - r <= b) bits |= braille.bit(r, 1);
    }
    out.push(braille.glyph(bits));
  }
  return out;
}
