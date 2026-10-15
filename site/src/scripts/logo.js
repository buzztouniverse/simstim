// The logo: three braille cells of a small wave, as in the launcher's
// header. Shared by the build (the still frame) and the page (the motion).

const BITS = [[0, 3], [1, 4], [2, 5], [6, 7]];

export function logo(t) {
	let s = '';
	for (let c = 0; c < 3; c++) {
		let bits = 0;
		for (let dx = 0; dx < 2; dx++) {
			const x = c * 2 + dx;
			const y = Math.round(1.5 + 1.5 * Math.sin(x * 0.9 - t * 4));
			bits |= 1 << BITS[Math.min(Math.max(y, 0), 3)][dx];
		}
		s += String.fromCharCode(0x2800 + bits);
	}
	return s;
}
