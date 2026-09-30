/** `[class to activate, class to restore]` pairs toggled on the button. */
type ClassPair = [on: string, off: string];

interface CopyOptions {
	pairs: ClassPair[];
	doneLabel?: string;
	duration?: number;
}

/**
 * Wires a copy-to-clipboard button: copies the text, flips the label to
 * "DONE", swaps in the accent classes, then reverts after `duration`.
 *
 * Fails silently when the Clipboard API is unavailable (insecure context)
 * or the read is rejected.
 */
export function attachCopy(
	btn: HTMLButtonElement,
	getText: () => string | null | undefined,
	{ pairs, doneLabel = 'DONE', duration = 2000 }: CopyOptions
): void {
	const idleLabel = btn.textContent ?? 'COPY';
	let timer: ReturnType<typeof setTimeout> | undefined;

	// Announce the label flip ("COPY" -> "DONE") to assistive tech.
	btn.setAttribute('aria-live', 'polite');

	btn.addEventListener('click', async () => {
		const text = getText();
		if (!text || !navigator.clipboard) return;

		try {
			await navigator.clipboard.writeText(text);
		} catch {
			return;
		}

		btn.textContent = doneLabel;
		for (const [on, off] of pairs) {
			btn.classList.remove(off);
			btn.classList.add(on);
		}

		clearTimeout(timer);
		timer = setTimeout(() => {
			btn.textContent = idleLabel;
			for (const [on, off] of pairs) {
				btn.classList.remove(on);
				btn.classList.add(off);
			}
		}, duration);
	});
}
