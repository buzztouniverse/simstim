// The landing page's script (decision 0022). The page reads the same
// without it. It adds the copy buttons and the live launcher. The hero's
// story has its own few lines (components/landing/HeroStory.astro).

import { Launcher } from './launcher.js';

const page = document.querySelector('[data-landing]');
const name = page?.dataset.name ?? '';

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

// A command too long for its box scrolls; it fades at the edge while there
// is more to see.
for (const code of document.querySelectorAll('.cmd code')) {
	const mark = () => {
		code.classList.toggle('over', code.scrollWidth > code.clientWidth + 1);
		code.classList.toggle('end', code.scrollLeft + code.clientWidth >= code.scrollWidth - 1);
	};
	code.addEventListener('scroll', mark, { passive: true });
	new ResizeObserver(mark).observe(code);
}

// ---------- The launcher ----------
// It starts when it first comes into view, plays its short boot, then
// holds still until the reader points at it or gives it focus.

const screen = document.querySelector('[data-term-screen]');
if (screen) {
	new IntersectionObserver((entries, io) => {
		if (!entries.some((e) => e.isIntersecting)) return;
		io.disconnect();
		const launcher = new Launcher(screen, { name, say });
		if (launcher.bootAt < 0) launcher.boot();
	}, { rootMargin: '200px 0px' }).observe(screen);
}
