import { themes, DEFAULT_THEME } from './themes';

export const THEME_KEY = 'gaur-theme';

const ids = themes.map((t) => t.id);
const listeners = new Set<(id: string) => void>();

const read = (): string => {
	try {
		const stored = localStorage.getItem(THEME_KEY);
		if (stored && ids.includes(stored)) return stored;
	} catch {
		/* storage disabled */
	}
	return document.documentElement.getAttribute('data-theme') || DEFAULT_THEME;
};

export const currentTheme = read;

export const setTheme = (id: string): string => {
	const next = ids.includes(id) ? id : DEFAULT_THEME;
	document.documentElement.setAttribute('data-theme', next);
	try {
		localStorage.setItem(THEME_KEY, next);
	} catch {
		/* storage disabled */
	}
	listeners.forEach((fn) => fn(next));
	return next;
};

export const onThemeChange = (fn: (id: string) => void): (() => void) => {
	listeners.add(fn);
	return () => listeners.delete(fn);
};
