/** A, B, … Z, AA, AB, … (0-based). */
export function outlineLetter(index0: number): string {
	if (index0 < 0) return '';
	let n = index0;
	let out = '';
	do {
		out = String.fromCharCode(65 + (n % 26)) + out;
		n = Math.floor(n / 26) - 1;
	} while (n >= 0);
	return out;
}

/** Drop server `Title ×N` suffix so the header can show a clean step name. */
export function displayStepTitle(title: string): string {
	return title.replace(/\s×\d+$/u, '').trim();
}
