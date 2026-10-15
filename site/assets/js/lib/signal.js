// Shared pieces of the site's motion: braille cells, the motion setting, and
// a frame loop that runs only while it should.

// Braille: the dot at row r (0-3) and column c (0-1) of a cell, as the
// launcher draws it (internal/tui/mark.go).
const BITS = [[0, 3], [1, 4], [2, 5], [6, 7]];
export const braille = {
  bit: (r, c) => 1 << BITS[r][c],
  glyph: (bits) => String.fromCharCode(0x2800 + bits),
};

// Motion: "auto" follows prefers-reduced-motion; "on" and "off" are the
// reader's choice, kept in this browser. The head sets data-motion first.
const reduce = window.matchMedia("(prefers-reduced-motion: reduce)");
const EVENT = "site:motion"; // an event, so every script on the page hears it

export function getMotion() {
  try {
    const m = localStorage.getItem("motion");
    if (m === "on" || m === "off") return m;
  } catch (e) {
    // storage may be blocked; auto it is
  }
  return "auto";
}

export function motionOn() {
  const m = getMotion();
  return m === "on" || (m === "auto" && !reduce.matches);
}

function changed() {
  document.documentElement.dataset.motion = motionOn() ? "on" : "off";
  window.dispatchEvent(new Event(EVENT));
}

export function setMotion(m) {
  try {
    if (m === "auto") localStorage.removeItem("motion");
    else localStorage.setItem("motion", m);
  } catch (e) {
    // not kept, but it still applies to this page
  }
  changed();
}

export function onMotionChange(f) {
  window.addEventListener(EVENT, f);
}

reduce.addEventListener("change", () => {
  if (getMotion() === "auto") changed();
});

// everyFrame calls draw(t) up to fps times a second, t in seconds, while run()
// is true, the tab is visible and, if el is given, el is on screen. When it
// stops for good (run() false), it calls still() once, for a still frame.
export function everyFrame(fps, draw, run, still, el) {
  let visible = !el;
  let raf = 0;
  let last = 0;
  let stilled = false;
  const tick = (now) => {
    raf = 0;
    if (!active()) return;
    if (now - last >= 1000 / fps - 2) {
      last = now;
      draw(now / 1000);
    }
    raf = requestAnimationFrame(tick);
  };
  const active = () => run() && !document.hidden && visible;
  const update = () => {
    if (!run()) {
      if (!stilled && still) still();
      stilled = true;
    } else {
      stilled = false;
    }
    if (active() && !raf) raf = requestAnimationFrame(tick);
  };
  document.addEventListener("visibilitychange", update);
  onMotionChange(update);
  if (el && "IntersectionObserver" in window) {
    new IntersectionObserver((entries) => {
      visible = entries[entries.length - 1].isIntersecting;
      update();
    }).observe(el);
  } else {
    visible = true;
  }
  update();
  return { update };
}
