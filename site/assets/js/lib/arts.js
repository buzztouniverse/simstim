// The landing page's small pieces of braille and text motion: the welcome
// mark, the little screens on the cards, and the packets on the diagram.
// Each one starts from what the page shows without scripts, moves only with
// motion on, and stops when it is off screen.

import { braille, motionOn, everyFrame } from "./signal.js";

const NOISE = "⡏⢸⣿⠿⡿⢿⣻⣽⣾⣷⣯⣟⡷⢾⣼⣧";
const ease = (x) => 1 - Math.pow(1 - Math.min(Math.max(x, 0), 1), 3);

// ---------- The welcome mark (internal/tui/mark.go) ----------

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

// ---------- The card screens ----------

function trace(events, t, cells = 8) {
  let out = "";
  const dots = cells * 2;
  for (let c = 0; c < cells; c++) {
    let bits = 0;
    for (let dx = 0; dx < 2; dx++) {
      const col = c * 2 + dx;
      const at = t - (24 * (dots - 1 - col)) / (dots - 1);
      let v = 0;
      for (const e of events) {
        const d = (at - e) / 1.4;
        v = Math.max(v, Math.exp(-d * d));
      }
      const h = 1 + Math.round(v * 3);
      for (let dy = 0; dy < h; dy++) bits |= braille.bit(3 - dy, dx);
    }
    out += braille.glyph(bits);
  }
  return out;
}

const span = (cls, text) => `<span class="${cls}">${text}</span>`;

function agentsArt(el, kinds) {
  if (!kinds.length) return;
  let i = 0;
  let at = 0;
  everyFrame(
    15,
    (t) => {
      if (!at) at = t;
      if (t - at > 2.4) {
        at = t;
        i = (i + 1) % kinds.length;
      }
      const k = decode(kinds[i], t - at);
      el.innerHTML = `❯ new my-vps --agent <b class="a">${k}</b>\n${span("f", "  ✓ started " + kinds[i] + " in a session")}`;
    },
    () => motionOn(),
    () => {
      el.innerHTML = `❯ new my-vps --agent <b class="a">claude</b>\n${span("f", "  ✓ started claude in a session")}`;
    },
    el,
  );
}

function hostsArt(el) {
  const hosts = [
    ["homelab", 4, 0.9],
    ["gcp-dev", 38, 1.7],
  ];
  const ev = hosts.map(() => []);
  everyFrame(
    10,
    (t) => {
      let html = "";
      hosts.forEach(([name, rtt, every], i) => {
        const e = ev[i];
        if (!e.length || t - e[e.length - 1] > every + ((t * 7) % 1) * 0.8) e.push(t);
        while (e.length && t - e[0] > 30) e.shift();
        const ms = Math.max(1, Math.round(rtt + Math.sin(t * 1.7 + i) * rtt * 0.15));
        html += `${span("o", "●")} ${name.padEnd(12)}${span("f", (ms + "ms").padStart(5))}  ${span("a", trace(e, t))}\n`;
      });
      html += `${span("f", "◌ hetzner-box  offline  ⣀⣀⣀⣀⣀⣀⣀⣀")}`;
      el.innerHTML = html;
    },
    () => motionOn(),
    null,
    el,
  );
}

const SEQUENCES = [
  ["shift+enter", "kitty keys", "1b 5b 31 33 3b 32 75"],
  ["a link", "OSC 8", "1b 5d 38 3b 3b 68 74 74 70 73"],
  ["copy", "OSC 52", "1b 5d 35 32 3b 63 3b 61 47 6b"],
  ["truecolour", "SGR 38;2", "1b 5b 33 38 3b 32 3b 30 3b 32"],
  ["notify", "OSC 777", "1b 5d 37 37 37 3b 6e 6f 74 69"],
];

function bytesArt(el) {
  let i = 0;
  let at = 0;
  everyFrame(
    20,
    (t) => {
      if (!at) at = t;
      const s = t - at;
      if (s > 3.2) {
        at = t;
        i = (i + 1) % SEQUENCES.length;
      }
      const [what, kind, hex] = SEQUENCES[i];
      const n = hex.length;
      const a = Math.min(n, Math.floor((s / 1.1) * n));
      const b = Math.min(n, Math.floor(((s - 0.35) / 1.1) * n));
      const ok = b >= n ? ` ${span("o", "✓ same bytes")}` : "";
      el.innerHTML =
        `${span("f", kind.padEnd(10))}  ${span("a", what)}\n` +
        `${span("f", "host        ")}${hex.slice(0, a)}\n` +
        `${span("f", "you         ")}${hex.slice(0, Math.max(b, 0))}${ok}`;
    },
    () => motionOn(),
    () => {
      const [what, kind, hex] = SEQUENCES[0];
      el.innerHTML = `${span("f", kind.padEnd(10))}  ${span("a", what)}\n${span("f", "host        ")}${hex}\n${span("f", "you         ")}${hex} ${span("o", "✓ same bytes")}`;
    },
    el,
  );
}

const SPIN = "⠁⠂⠄⡀⢀⠠⠐⠈";

function sleepArt(el) {
  const ev = [];
  let at = 0;
  everyFrame(
    12,
    (t) => {
      if (!at) at = t;
      const s = (t - at) % 9;
      const online = s < 3.2 || s > 5.4;
      if (online && (!ev.length || t - ev[ev.length - 1] > 0.8)) ev.push(t);
      while (ev.length && t - ev[0] > 30) ev.shift();
      let status;
      if (s < 3.2) status = `${span("a", "▸")} ${span("f", "working: 14 files changed")}`;
      else if (s < 4.4) status = `${span("f", "◌ laptop asleep, link down")}`;
      else if (s < 5.4) status = `${span("a", SPIN[Math.floor(t * 11) % SPIN.length])} ${span("f", "reconnecting")}`;
      else status = `${span("o", "✓")} ${span("f", "back: replayed " + (12 + Math.floor((at * 10) % 9)) + " KiB you missed")}`;
      el.innerHTML = `${span("a", trace(ev, t, 14))}\n${status}`;
    },
    () => motionOn(),
    null,
    el,
  );
}

export function cardArts(kinds) {
  const get = (n) => document.querySelector(`[data-art="${n}"]`);
  if (get("agents")) agentsArt(get("agents"), kinds);
  if (get("hosts")) hostsArt(get("hosts"));
  if (get("bytes")) bytesArt(get("bytes"));
  if (get("sleep")) sleepArt(get("sleep"));
}

// ---------- The diagram ----------

export function diagram(fig, parts) {
  if (!fig || !parts) return;
  const buttons = [...parts.querySelectorAll("button[data-part]")];
  let pinned = null;
  const light = (part) => {
    for (const s of fig.querySelectorAll("[data-part]")) {
      const p = s.dataset.part;
      s.classList.toggle("lit", !!part && (p === part || (part === "ssh" && p.startsWith("ssh")) || (part === "holder" && p === "agent")));
    }
  };
  for (const b of buttons) {
    b.addEventListener("pointerenter", () => light(b.dataset.part));
    b.addEventListener("pointerleave", () => light(pinned));
    b.addEventListener("focus", () => light(b.dataset.part));
    b.addEventListener("blur", () => light(pinned));
    b.addEventListener("click", () => {
      pinned = pinned === b.dataset.part ? null : b.dataset.part;
      for (const o of buttons) o.setAttribute("aria-pressed", String(o.dataset.part === pinned));
      light(pinned);
    });
  }
  // Packets on the ssh arrows: keys going out, the screen coming back.
  const outs = [...fig.querySelectorAll('[data-part="ssh-out"]')].filter((s) => s.textContent.length > 2);
  const ins = [...fig.querySelectorAll('[data-part="ssh-in"]')].filter((s) => s.textContent.length > 2);
  const base = new Map([...outs, ...ins].map((s) => [s, s.textContent]));
  everyFrame(
    14,
    (t) => {
      for (const s of outs) {
        const b = base.get(s);
        const i = Math.floor(t * 9) % (b.length - 1);
        s.textContent = b.slice(0, i) + "⠶" + b.slice(i + 1);
      }
      for (const s of ins) {
        const b = base.get(s);
        const n = b.length - 1;
        const i = n - (Math.floor(t * 12 + 3) % n);
        s.textContent = b.slice(0, i) + "⣿" + b.slice(i + 1);
      }
    },
    () => motionOn(),
    () => {
      for (const [s, b] of base) s.textContent = b;
    },
    fig,
  );
}
