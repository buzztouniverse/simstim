// Shared pieces of the site's motion: braille cells, whether motion is on,
// and a frame loop that runs only while it should. There is no switch on
// the page: motion follows the reader's system setting
// (prefers-reduced-motion), and it changes as soon as that setting does.

export { braille } from "./braille.js";

const reduce = window.matchMedia("(prefers-reduced-motion: reduce)");

export function motionOn() {
  return !reduce.matches;
}

export function onMotionChange(f) {
  reduce.addEventListener("change", f);
}

// everyFrame calls draw(t) up to fps times a second, t in seconds, while run()
// is true, the tab is visible and, if el is given, el is on screen. When it
// stops for good (run() false), it calls still() once, for a still frame.
// Call update() when run() may have changed.
export function everyFrame(fps, draw, run, still, el) {
  let visible = !el;
  let raf = 0;
  let last = 0;
  let stilled = false;
  const active = () => run() && !document.hidden && visible;
  const tick = (now) => {
    raf = 0;
    if (!active()) return;
    if (now - last >= 1000 / fps - 2) {
      last = now;
      draw(now / 1000);
    }
    raf = requestAnimationFrame(tick);
  };
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
