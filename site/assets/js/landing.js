// The landing page's script (decision 0021): the live launcher, the copy
// buttons, the small animations, and the three.js signal field, which loads
// last and only when motion is on, WebGL works and the reader has not asked
// to save data. The page reads the same without any of it.

import { motionOn, onMotionChange, setMotion } from "./lib/signal.js";
import { Launcher } from "./lib/launcher.js";
import { animateMark, cardArts, diagram } from "./lib/arts.js";

const page = document.querySelector(".landing");
const name = page.dataset.name;

// A polite voice for screen readers: "Copied", and the like.
const voice = document.createElement("div");
voice.className = "sr-only";
voice.setAttribute("aria-live", "polite");
document.body.append(voice);
const say = (text) => {
  voice.textContent = "";
  requestAnimationFrame(() => (voice.textContent = text));
};

for (const el of document.querySelectorAll(".js-only")) el.hidden = false;

// ---------- Pause ----------
// A visible switch for everything that moves on this page (WCAG 2.2.2). It
// sets the same choice as the footer's motion switch.

const pause = document.querySelector("[data-motion-toggle]");
if (pause) {
  const show = () => {
    const on = motionOn();
    pause.textContent = on ? "‖ pause the motion" : "▶ play the motion";
    pause.setAttribute("aria-pressed", String(!on));
  };
  pause.addEventListener("click", () => {
    setMotion(motionOn() ? "off" : "on");
    say(motionOn() ? "Motion on." : "Motion paused.");
  });
  onMotionChange(show);
  show();
}

// ---------- Copy buttons ----------

for (const box of document.querySelectorAll("[data-copy-box]")) {
  const btn = box.querySelector("[data-copy]");
  const text = box.querySelector("[data-copy-text]");
  const label = box.querySelector("[data-copy-label]");
  if (!btn || !text) continue;
  btn.addEventListener("click", async () => {
    const value = text.textContent.trim();
    let ok = false;
    try {
      await navigator.clipboard.writeText(value);
      ok = true;
    } catch (e) {
      const r = document.createRange();
      r.selectNodeContents(text);
      const sel = getSelection();
      sel.removeAllRanges();
      sel.addRange(r);
      ok = document.execCommand && document.execCommand("copy");
    }
    label.textContent = ok ? "copied ✓" : "select it";
    say(ok ? "Copied." : "Copy did not work. The text is selected; copy it with your keyboard.");
    setTimeout(() => (label.textContent = "copy"), 1600);
  });
}

// ---------- The launcher ----------

const screen = document.querySelector("[data-term-screen]");
if (screen) {
  const launcher = new Launcher(screen, { name, say });
  // It plays its boot sequence when it first comes into view. Until then it
  // is drawn in full, so a page read without scrolling still shows it all.
  new IntersectionObserver((entries, io) => {
    if (entries.some((e) => e.isIntersecting)) {
      if (launcher.bootAt < 0) launcher.boot();
      io.disconnect();
    }
  }).observe(screen);
}

// ---------- Small animations ----------

const kinds = [...document.querySelectorAll(".agent-group code")].map((c) => c.textContent.replace("--agent ", "")).filter((k) => k !== "shell");
cardArts(kinds.length ? kinds : ["claude", "codex", "opencode"]);
diagram(document.querySelector("[data-diagram]"), document.querySelector(".parts"));
animateMark(document.querySelector("[data-mark]"), document.querySelector("[data-decode]"));

// ---------- The signal field (three.js) ----------

function webgl() {
  try {
    const c = document.createElement("canvas");
    const o = { failIfMajorPerformanceCaveat: true };
    return !!(window.WebGLRenderingContext && (c.getContext("webgl2", o) || c.getContext("webgl", o)));
  } catch (e) {
    return false;
  }
}

const host = document.querySelector("[data-field-host]");
const still = host && host.querySelector(".field-still");
let field = null;
let loading = false;
let fallback = false;
let onScreen = false;

function saveData() {
  return (navigator.connection && navigator.connection.saveData) || matchMedia("(prefers-reduced-data: reduce)").matches;
}

// three.js costs a phone's processor more than the field is worth there:
// phones and tablets get the braille ribbon, moving, instead.
const wide = matchMedia("(min-width: 720px) and (pointer: fine)");

// The field loads when motion is on, the stage is on screen, the reader has
// not asked to save data, the screen is wide with a mouse or trackpad, and
// WebGL runs on the GPU. Otherwise the braille mark stays: still, or, with
// motion on, moving as it does on the launcher's welcome screen.
async function loadField() {
  if (field || loading || fallback || !host || !motionOn() || !onScreen) return;
  if (saveData() || !wide.matches || !webgl()) {
    fallback = true;
    animateMark(still);
    return;
  }
  loading = true;
  try {
    const m = await import(page.dataset.field);
    field = m.start(host);
    if (!field) {
      fallback = true;
      animateMark(still);
    }
  } catch (e) {
    fallback = true;
    animateMark(still);
  }
  loading = false;
}

function later() {
  if ("requestIdleCallback" in window) requestIdleCallback(loadField, { timeout: 2500 });
  else setTimeout(loadField, 1200);
}

if (host) {
  new IntersectionObserver((entries) => {
    onScreen = entries[entries.length - 1].isIntersecting;
    if (onScreen && document.readyState === "complete") later();
  }, { rootMargin: "200px" }).observe(host);
  if (document.readyState !== "complete") window.addEventListener("load", later, { once: true });
  onMotionChange(() => motionOn() && later());
  // Free the GPU when the page goes away for good.
  window.addEventListener("pagehide", (e) => {
    if (!e.persisted && field) {
      field.stop();
      field = null;
    }
  });
}
