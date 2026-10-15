// The script every page shares: the logo's braille wave, the motion switch,
// and "view source" links. The landing page adds its own (landing.js).

import { braille, motionOn, onMotionChange, setMotion, getMotion, everyFrame } from "./lib/signal.js";

// The logo: three braille cells of a small wave, as in the launcher's header.
function logo(t) {
  let s = "";
  for (let c = 0; c < 3; c++) {
    let bits = 0;
    for (let dx = 0; dx < 2; dx++) {
      const x = c * 2 + dx;
      const y = Math.round(1.5 + 1.5 * Math.sin(x * 0.9 - t * 4));
      bits |= braille.bit(Math.min(Math.max(y, 0), 3), dx);
    }
    s += braille.glyph(bits);
  }
  return s;
}

function startLogos() {
  const marks = [...document.querySelectorAll("[data-logo]")];
  if (!marks.length) return;
  const draw = (t) => {
    const s = logo(t);
    for (const m of marks) {
      if (m.textContent !== s) m.textContent = s;
    }
  };
  draw(0);
  everyFrame(12, draw, () => motionOn(), () => draw(0));
}

// The motion switch in the footer: auto, on or off, kept in this browser.
function motionSwitch() {
  for (const box of document.querySelectorAll("[data-motion-switch]")) {
    box.hidden = false;
    const buttons = [...box.querySelectorAll("button[data-motion]")];
    const show = () => {
      const cur = getMotion();
      for (const b of buttons) b.setAttribute("aria-pressed", String(b.dataset.motion === cur));
    };
    for (const b of buttons) b.addEventListener("click", () => setMotion(b.dataset.motion));
    onMotionChange(show);
    show();
  }
}

// On a phone, Hextra hides the closed menu from screen readers with
// aria-hidden; inert also takes its links out of the Tab order.
function inertMenu() {
  const menu = document.querySelector(".hextra-sidebar-container");
  if (!menu) return;
  const sync = () => {
    menu.inert = menu.getAttribute("aria-hidden") === "true";
  };
  new MutationObserver(sync).observe(menu, { attributes: true, attributeFilter: ["aria-hidden"] });
  // Before Hextra opens the menu and moves focus into it, let it take focus.
  document.addEventListener("click", (e) => {
    if (e.target.closest(".hextra-hamburger-menu")) menu.inert = false;
  }, true);
  sync();
}

startLogos();
motionSwitch();
inertMenu();
