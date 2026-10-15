// The landing page's script (decision 0022). Every figure is drawn in full
// in the HTML, so the page reads the same with no script and with motion
// off. This adds: the live launcher; the stories of figures 2 and 3, each
// played once when it comes into view, step by step from its list, and
// again from "Play again"; the copy buttons; the pause switch; and the
// welcome mark at the end.

import { motionOn, onMotionChange, setMotion } from './signal.js';
import { Launcher } from './launcher.js';
import { animateMark } from './mark.js';
import { NOISE } from './braille.js';

const page = document.querySelector('[data-landing]');
const name = page?.dataset.name ?? '';
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// A polite voice for screen readers: "Copied", and the like.
const voice = document.createElement('div');
voice.className = 'sr-only';
voice.setAttribute('aria-live', 'polite');
document.body.append(voice);
const say = (text) => {
	voice.textContent = '';
	requestAnimationFrame(() => (voice.textContent = text));
};

for (const el of document.querySelectorAll('[data-js-only]')) el.hidden = false;

// ---------- Pause ----------
// One switch for everything that moves on the page (WCAG 2.2.2). It is the
// same setting as the footer's motion switch.

for (const pause of document.querySelectorAll('[data-motion-toggle]')) {
	const show = () => {
		const on = motionOn();
		pause.textContent = on ? 'Pause motion' : 'Play motion';
		pause.setAttribute('aria-pressed', String(!on));
	};
	pause.addEventListener('click', () => {
		setMotion(motionOn() ? 'off' : 'on');
		say(motionOn() ? 'Motion on.' : 'Motion paused.');
	});
	onMotionChange(show);
	show();
}

// ---------- Copy buttons ----------

for (const box of document.querySelectorAll('[data-copy-box]')) {
	const btn = box.querySelector('[data-copy]');
	const text = box.querySelector('[data-copy-text]');
	if (!btn || !text) continue;
	btn.addEventListener('click', async () => {
		let ok = false;
		try {
			await navigator.clipboard.writeText(text.textContent.trim());
			ok = true;
		} catch {
			const r = document.createRange();
			r.selectNodeContents(text);
			getSelection().removeAllRanges();
			getSelection().addRange(r);
		}
		btn.textContent = ok ? 'Copied' : 'Selected';
		say(ok ? 'Copied.' : 'Copy did not work. The text is selected; copy it with your keyboard.');
		setTimeout(() => (btn.textContent = 'Copy'), 1600);
	});
}

// ---------- Fig. 1: the launcher ----------

const screen = document.querySelector('[data-term-screen]');
if (screen) {
	const launcher = new Launcher(screen, { name, say });
	// It plays its boot sequence when it first comes into view. Until then
	// it is drawn in full, so a page read without scrolling shows it all.
	new IntersectionObserver((entries, io) => {
		if (entries.some((e) => e.isIntersecting)) {
			if (launcher.bootAt < 0) launcher.boot();
			io.disconnect();
		}
	}).observe(screen);
}

// ---------- Stories ----------
// A story is a figure with an ordered list of steps beside it. Each step is
// a button that shows the figure at that step; with motion on, the figure
// plays through the steps once when it is mostly in view, and from "Play
// again". With motion off, the figure jumps from step to step.

function story(fig, steps) {
	const buttons = [...fig.querySelectorAll('[data-step-to]')];
	const replay = fig.querySelector('[data-replay]');
	let run = 0;
	const mark = (n) => {
		fig.dataset.step = String(n);
		for (const b of buttons) b.setAttribute('aria-pressed', String(Number(b.dataset.stepTo) === n));
	};
	const show = async (n, animate) => {
		const id = ++run;
		const stop = () => id !== run || !motionOn() || document.hidden;
		mark(n);
		await steps[n - 1](animate && motionOn() ? stop : () => true);
	};
	const play = async () => {
		const id = ++run;
		for (let n = 1; n <= steps.length; n++) {
			if (id !== run) return;
			mark(n);
			await steps[n - 1](() => id !== run || !motionOn() || document.hidden);
			if (id !== run) return;
			if (!motionOn()) break;
			await sleep(500);
		}
		if (id === run) await show(steps.length, false);
	};
	for (const b of buttons) b.addEventListener('click', () => show(Number(b.dataset.stepTo), true));
	if (replay) replay.addEventListener('click', () => motionOn() ? play() : show(steps.length, false));
	let played = false;
	new IntersectionObserver((entries) => {
		if (!played && motionOn() && entries.some((e) => e.intersectionRatio >= 0.6)) {
			played = true;
			play();
		}
	}, { threshold: [0, 0.6, 1] }).observe(fig);
	onMotionChange(() => {
		if (!motionOn()) {
			run++;
			show(Number(fig.dataset.step) || steps.length, false);
		}
	});
}

// Fig. 2: two traces over forty minutes. The host's never stops; the
// laptop's goes quiet while the lid is shut, then the missed output
// arrives in one burst. The figure has two renditions (a cell a minute,
// and a cell per two minutes for phones); the story drives both, in
// fractions of the forty minutes.
function sleepFigure(fig) {
	if (!fig) return;
	const traces = [...fig.querySelectorAll('[data-trace]')].map((tr) => {
		const lap = [...tr.querySelectorAll('[data-laptop] > span')];
		return {
			host: [...tr.querySelectorAll('[data-host] > span')],
			lap,
			text: lap.map((c) => c.textContent),
			head: tr.querySelector('[data-head]'),
			close: Number(tr.dataset.close),
			open: Number(tr.dataset.open),
			end: Number(tr.dataset.open) + Number(tr.dataset.burst),
		};
	});
	// The figure at time f (0 to 1): what has been drawn by then. The missed
	// output shows on the laptop only once it has been replayed.
	const render = (f, replayed) => {
		for (const t of traces) {
			const i = Math.round(f * t.host.length);
			t.head.style.setProperty('--x', String(i));
			t.host.forEach((c, k) => c.classList.toggle('off', k >= i));
			t.lap.forEach((c, k) => {
				const missed = k >= t.open && k < t.end;
				c.classList.toggle('off', missed ? !replayed : k >= i);
			});
		}
	};
	const at = (name) => {
		const t = traces[0];
		return { close: t.close / t.host.length, open: t.open / t.host.length, end: t.end / t.host.length }[name];
	};
	const sweep = async (from, to, replayed, stop) => {
		fig.classList.add('sweeping');
		const steps = Math.max(1, Math.round((to - from) * 40));
		for (let k = 0; k <= steps && !stop(); k++) {
			render(from + ((to - from) * k) / steps, replayed);
			await sleep(110);
		}
		fig.classList.remove('sweeping');
	};
	story(fig, [
		// 1. Attached: both traces run together.
		async (stop) => {
			if (!stop()) await sweep(0, at('close'), false, stop);
			render(at('close'), false);
		},
		// 2. The lid shuts: the laptop goes quiet; the host goes on.
		async (stop) => {
			if (!stop()) await sweep(at('close'), at('open'), false, stop);
			render(at('open'), false);
		},
		// 3. Back: the missed output replays at once, as a burst.
		async (stop) => {
			render(at('open'), false);
			for (let k = 0; k < 8 && !stop(); k++) {
				for (const t of traces) {
					for (let j = t.open; j < t.end; j++) {
						t.lap[j].classList.remove('off');
						t.lap[j].textContent = NOISE[(j * 5 + k * 3) % NOISE.length];
					}
				}
				await sleep(55);
			}
			for (const t of traces) t.lap.forEach((c, k) => (c.textContent = t.text[k]));
			render(at('end'), true);
		},
		// 4. Caught up: both traces run together again.
		async (stop) => {
			if (!stop()) await sweep(at('end'), 1, true, stop);
			render(1, true);
		},
	]);
}

// Fig. 3: the same bytes. Shift+Enter's seven bytes land one by one on
// your side, then on the agent's through the client, and through tmux.
function bytesFigure(fig) {
	if (!fig) return;
	const rows = [...fig.querySelectorAll('[data-bytes]')];
	const cells = rows.map((r) => [...r.querySelectorAll('[data-b]')]);
	const verdicts = [...fig.querySelectorAll('[data-verdict]')];
	const drawRow = async (r, stop) => {
		for (const c of cells[r]) c.classList.add('off');
		if (verdicts[r]) verdicts[r].classList.add('off');
		for (const c of cells[r]) {
			if (stop()) break;
			c.classList.remove('off');
			c.classList.add('land');
			setTimeout(() => c.classList.remove('land'), 240);
			await sleep(90);
		}
		for (const c of cells[r]) c.classList.remove('off');
		if (verdicts[r]) verdicts[r].classList.remove('off');
	};
	const upto = (r) => {
		cells.forEach((row, k) => row.forEach((c) => c.classList.toggle('off', k > r)));
		verdicts.forEach((v, k) => v.classList.toggle('off', k > r));
	};
	story(fig, [
		async (stop) => { upto(-1); await drawRow(0, stop); upto(0); },
		async (stop) => { upto(0); await drawRow(1, stop); upto(1); },
		async (stop) => { upto(1); await drawRow(2, stop); upto(2); },
	]);
}

sleepFigure(document.querySelector('[data-sleep-fig]'));

// Fig. 4: a pulse runs along the ssh link while the figure is on screen.
for (const wire of document.querySelectorAll('[data-wire]')) {
	new IntersectionObserver((entries) => {
		wire.classList.toggle('live', entries[entries.length - 1].isIntersecting);
	}).observe(wire);
}
bytesFigure(document.querySelector('[data-bytes-fig]'));

// ---------- The end: the welcome mark ----------

animateMark(document.querySelector('[data-mark]'), document.querySelector('[data-decode]'));
