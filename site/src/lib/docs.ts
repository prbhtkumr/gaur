import type { CollectionEntry } from 'astro:content';

/** Display order for the docs guides. Anything not listed sorts alphabetically after. */
const ORDER: readonly string[] = ['installation', 'usage', 'configuration', 'internals'];

/** Returns a new array ordered for display. Never mutates the input. */
export function sortDocs(entries: CollectionEntry<'docs'>[]): CollectionEntry<'docs'>[] {
	return [...entries].sort((a, b) => {
		const ai = ORDER.indexOf(a.id);
		const bi = ORDER.indexOf(b.id);
		if (ai !== -1 && bi !== -1) return ai - bi;
		if (ai !== -1) return -1;
		if (bi !== -1) return 1;
		return a.data.title.localeCompare(b.data.title);
	});
}
