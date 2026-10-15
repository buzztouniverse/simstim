// The welcome mark (internal/tui/mark.go): three braille strands that open
// out and flow, and the name decoding out of braille noise. The landing page
// ends with it. It starts from the still frame the page is built with,
// moves only with motion on, and stops when it is off screen.

import { motionOn, everyFrame } from "./signal.js";
import { braille, markLines, NOISE } from "./braille.js";

const ease = (x) => 1 - Math.pow(1 - Math.min(Math.max(x, 0), 1), 3);

// animateMark plays the welcome on a <pre>: the ribbon opens out from its
// middle, then flows, and a pulse runs along it. With name, the name decodes
// out of braille noise after the ribbon starts to open.
export function animateMark(pre, nameEl) {
  if (!pre) return;
  const text = nameEl ? nameEl.textContent : "";
  let start = -1;
  const draw = (t) => {
    if (start < 0) start = t;
    const s = t - start;
    const grow = ease(s / 0.9);
    const lines = markLines(s, grow);
    const pulse = -5 + ((s / 2.6) % 1) * 54;
    let html = "";
    for (const row of lines) {
      let line = "";
      row.forEach(([r, n], cx) => {
        if (!r) {
          line += " ";
          return;
        }
        const p = (cx - pulse) / 4;
        const k = Math.min(1, 0.4 + 0.1 * n + 0.55 * Math.exp(-p * p));
        line += `<span style="opacity:${k.toFixed(2)}">${braille.glyph(r)}</span>`;
      });
      html += line + "\n";
    }
    pre.innerHTML = html.replace(/\n$/, "");
    if (nameEl) nameEl.textContent = decode(text, s - 0.3);
  };
  const still = () => {
    const lines = markLines(0, 1);
    pre.textContent = lines.map((row) => row.map(([r]) => (r ? braille.glyph(r) : " ")).join("")).join("\n");
    if (nameEl) nameEl.textContent = text;
    start = -1;
  };
  everyFrame(30, draw, () => motionOn(), still, pre);
}

export function decode(text, s) {
  if (s >= 0.42) return text;
  const frame = Math.floor(performance.now() / 33);
  let out = "";
  for (let i = 0; i < text.length; i++) {
    const settle = (0.42 * (i + 1)) / (text.length + 1);
    if (text[i] === " " || s >= settle) out += text[i];
    else if (s < settle - 0.21) out += " ";
    else out += NOISE[(i * 7 + frame * 13) % NOISE.length];
  }
  return out;
}
