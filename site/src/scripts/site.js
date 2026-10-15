// The script every page shares: the logo's braille wave, and the motion
// switch in the footer. The landing page adds its own (landing.js).

import { motionOn, onMotionChange, setMotion, getMotion, everyFrame } from './signal.js';
import { logo } from './logo.js';

function startLogos() {
	const marks = [...document.querySelectorAll('[data-logo]')];
	if (!marks.length) return;
	const draw = (t) => {
		const s = logo(t);
		for (const m of marks) if (m.textContent !== s) m.textContent = s;
	};
	everyFrame(12, draw, () => motionOn(), () => draw(0), marks[0]);
}

// The motion switch: auto, on or off, kept in this browser.
function motionSwitch() {
	for (const box of document.querySelectorAll('[data-motion-switch]')) {
		box.hidden = false;
		const buttons = [...box.querySelectorAll('button[data-motion]')];
		const show = () => {
			const cur = getMotion();
			for (const b of buttons) b.setAttribute('aria-pressed', String(b.dataset.motion === cur));
		};
		for (const b of buttons) b.addEventListener('click', () => setMotion(b.dataset.motion));
		onMotionChange(show);
		show();
	}
}

startLogos();
motionSwitch();
