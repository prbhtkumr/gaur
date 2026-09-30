import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const themesDir = path.resolve(here, '../../themes');
const outCss = path.resolve(here, '../src/lib/themeStyles.ts');
const outTs = path.resolve(here, '../src/lib/themes.ts');

const WHITE = [255, 255, 255];
const BLACK = [0, 0, 0];

const lin = (c) => {
	const v = c / 255;
	return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
};
const lum = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
const contrast = (a, b) => {
	const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
};
const mix = (a, b, k) => a.map((v, i) => v * (1 - k) + b[i] * k);
const over = (fg, alpha, bg) => fg.map((v, i) => v * alpha + bg[i] * (1 - alpha));
const hex = (rgb) => '#' + rgb.map((v) => Math.round(v).toString(16).padStart(2, '0')).join('');
const chan = (rgb) => rgb.map((v) => Math.round(v)).join(' ');
// CSS stores integer channels only, so the gate must judge what it will emit.
const rd = (rgb) => rgb.map((v) => Math.round(v));

const parseToml = (src) => {
	const out = {};
	const re = /^\s*([a-z0-9_]+)\s*=\s*"(#[0-9a-fA-F]{6})"\s*(?:#.*)?$/;
	for (const line of src.split('\n')) {
		const m = re.exec(line);
		if (m) out[m[1]] = m[2].slice(1).match(/../g).map((h) => parseInt(h, 16));
	}
	return out;
};

// Raise a colour toward white until every (alpha, floor) constraint holds.
// Contrast rises monotonically with k for light-on-dark, so binary search is safe.
const ensure = (rawFg, rawBg, constraints) => {
	const fg = rd(rawFg);
	const bg = rd(rawBg);
	const ok = (k) =>
		constraints.every(([alpha, target]) => contrast(over(rd(mix(fg, WHITE, k)), alpha, bg), bg) >= target);
	if (ok(0)) return { color: fg, lift: 0 };
	if (!ok(1)) return { color: [255, 255, 255], lift: 1, unreachable: true };
	let lo = 0;
	let hi = 1;
	for (let i = 0; i < 40; i++) {
		const mid = (lo + hi) / 2;
		if (ok(mid)) hi = mid;
		else lo = mid;
	}
	return { color: rd(mix(fg, WHITE, hi)), lift: hi };
};

/* Site accent, per theme.
   Catppuccin's `selected` is mauve, which is exactly right for the three
   Catppuccin palettes — but every other theme ships a purple selection too,
   so routing the site accent through `selected` painted gruvbox, solarized
   and one-dark the same lavender as mocha. Each theme instead names the
   TOML colour that carries its own identity. */
const ACCENT_SRC = {
	'catppuccin-mocha': 'selected', // #cba6f7 mauve
	'catppuccin-frappe': 'selected', // #ca9ee6 mauve
	'catppuccin-macchiato': 'selected', // #c6a0f6 mauve
	dracula: 'dashboard', // #ff79c6
	'gruvbox-dark': 'multilib', // #fe8019
	'monokai-pro': 'dashboard', // #ff6188
	'one-dark': 'accent', // #61afef
	'rose-pine': 'remove', // #eb6f92
	'solarized-dark': 'accent', // #268bd2
	'tokyonight-night': 'accent', // #7aa2f7
	'tokyonight-storm': 'accent', // #7aa2f7
};

// The accent tints its own background (the picker's checked row), so that
// shell moves with the colour under test. Solve for a fixed point instead of
// checking once against a shell built from the raw, unlifted value — otherwise
// each lift quietly invalidates the shell that justified it.
// Returns the lift needed, 0 if none, or null if it is unreachable.
const ensureTint = (rawFg, bg, alpha, target) => {
	const ok = (k) => {
		const c = rd(mix(rawFg, WHITE, k));
		// The tint is a live composite in the DOM, not a stored channel value,
		// so it is judged unrounded — rounding it here reads a hair optimistic.
		return contrast(c, over(c, alpha, bg)) >= target;
	};
	if (ok(0)) return 0;
	if (!ok(1)) return null;
	let lo = 0;
	let hi = 1;
	for (let i = 0; i < 40; i++) {
		const mid = (lo + hi) / 2;
		if (ok(mid)) hi = mid;
		else lo = mid;
	}
	return hi;
};

const derive = (t, id) => {
	const bg = t.scrollbar_track;
	const rawText = t.text;
	const base = rd(mix(bg, WHITE, 0.03));
	const mantle = bg;
	const crust = rd(mix(bg, BLACK, 0.22));
	const surface0 = rd(mix(base, rawText, 0.1));
	const surface1 = rd(mix(base, rawText, 0.2));
	const surface2 = rd(mix(base, rawText, 0.3));

	// Site neutrals mirror Catppuccin's own relationships, so the default
	// theme reproduces today's palette before any contrast lift is applied.
	const mutedRaw = rd(over(rawText, 0.78, bg));
	const overlayRaw = rd(over(rawText, 0.7, bg));

	// (surface, text alpha, minimum contrast) triples that actually occur
	// in the templates — a brute-force matrix over every pairing would
	// over-lift colours and flatten the hierarchy.
	// Real composite surfaces: the focused package row paints surface1 at 20%
	// over the terminal, and feature cards paint mantle at 50% over the page.
	const row = rd(mix(base, surface1, 0.2));
	const card = rd(mix(base, mantle, 0.5));

	const sources = {
		text: rawText,
		muted: mutedRaw,
		overlay2: overlayRaw,
		mauve: t.selected,
		primary: t[ACCENT_SRC[id] ?? 'selected'],
		green: t.success,
		highlight: t.dashboard_value,
		pink: t.dashboard,
		blue: t.install,
		yellow: t.warning,
		peach: t.multilib,
		sapphire: t.accent,
	};

	// The active docs row sits on white/7 over the frosted rail; the accent's
	// own tinted row is handled separately by ensureTint below.
	const railRow = rd(over(WHITE, 0.07, rd(over(mantle, 0.55, base))));

	const rules = {
		text: { shells: [base, mantle, crust, surface0, row], needs: [[1, 7], [0.9, 4.5], [0.7, 4.5]] },
		muted: { shells: [base, mantle, crust, card], needs: [[1, 4.8], [0.8, 4.5]] },
		overlay2: { shells: [base, row, surface0, crust], needs: [[1, 4.5]] },
		mauve: { shells: [base, mantle, crust, card], needs: [[1, 5], [0.8, 4.5]] },
		// `primary` only ever paints full-opacity text (the alpha variants are
		// borders, underlines and the caret), so the threshold is the plain
		// text one — checked against every surface listed above as well.
		primary: {
			shells: [base, mantle, crust, card, railRow],
			needs: [[1, 4.5]],
		},
		green: { shells: [crust, base, row], needs: [[1, 7], [0.9, 4.5]] },
		highlight: { shells: [base, mantle, card, surface0], needs: [[1, 4.5]] },
		pink: { shells: [surface0, base, row], needs: [[1, 4.5]] },
		blue: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
		yellow: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
		peach: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
		sapphire: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
	};

	const fixes = {};
	const lifts = {};

	const palette = {
		base,
		mantle,
		crust,
		surface0,
		surface1,
		surface2,
	};
	for (const [key, { shells, needs }] of Object.entries(rules)) {
		const fg = sources[key];
		let best = { color: fg, lift: 0 };
		for (const shell of shells) {
			const r = ensure(fg, shell, needs);
			if (r.unreachable) {
				fixes[key] = 1;
				break;
			}
			if (r.lift > best.lift) best = r;
		}
		lifts[key] = best.lift;
		if (best.lift > 0) fixes[key] = best.lift;
		palette[key] = best.color;
	}

	// Fold the accent's self-tint constraint back in. Every shell in the rule
	// above only gets easier as the accent lightens, so the larger lift wins.
	const tintLift = ensureTint(sources.primary, base, 0.14, 4.5);
	if (tintLift === null) {
		fixes.primary = 1;
	} else if (tintLift > lifts.primary) {
		palette.primary = rd(mix(sources.primary, WHITE, tintLift));
		lifts.primary = tintLift;
		if (tintLift > 0) fixes.primary = tintLift;
	}

	return { palette, fixes };
};

const ALIASES = {
	background: 'base',
	surface: 'mantle',
	subtext0: 'muted',
};

const files = fs.readdirSync(themesDir).filter((f) => f.endsWith('.toml')).sort();
const themes = files.map((f) => {
	const id = f.replace(/\.toml$/, '').replace(/[_-]/g, '-');
	const toml = parseToml(fs.readFileSync(path.join(themesDir, f), 'utf8'));
	for (const key of ['scrollbar_track', 'text', 'selected', 'success', 'dashboard_value', 'dashboard', 'install', 'warning', 'multilib', 'accent']) {
		if (!toml[key]) throw new Error(`${f}: missing "${key}"`);
	}
	const { palette, fixes } = derive(toml, id);
	return {
		id,
		label: f.replace(/\.toml$/, '').replace(/[_-]/g, ' '),
		palette,
		fixes,
	};
});

const problems = [];
for (const t of themes) {
	if (lum(t.palette.base) > 0.18) problems.push(`${t.id}: background too light for dark UI (L=${lum(t.palette.base).toFixed(3)})`);
	for (const [k, v] of Object.entries(t.fixes)) {
		if (v >= 1) problems.push(`${t.id}: cannot lift "${k}" to a readable contrast`);
	}
}
if (problems.length) {
	console.error('theme palette gate failed:\n  ' + problems.join('\n  '));
	process.exit(1);
}

const keys = Object.keys(themes[0].palette);
const decl = (p) => keys.map((k) => `\t--c-${k}: ${chan(p[k])};`).join('\n');
const aliasDecl = Object.entries(ALIASES).map(([k, v]) => `\t--c-${k}: var(--c-${v});`).join('\n');

const css = [
	'/* generated by scripts/build-themes.mjs - do not edit */',
	':root {',
	decl(themes.find((t) => t.id === 'catppuccin-mocha').palette),
	aliasDecl,
	'}',
	...themes
		.filter((t) => t.id !== 'catppuccin-mocha')
		.map((t) => `html[data-theme="${t.id}"] {\n${decl(t.palette)}${aliasDecl}\n}`),
	'',
].join('\n');

const ts = [
	'/* generated by scripts/build-themes.mjs - do not edit */',
	"export interface SiteTheme {",
	'\tid: string;',
	'\tlabel: string;',
	'\tdot: { bg: string; fg: string; accent: string };',
	'}',
	'',
	'export const DEFAULT_THEME = ' + JSON.stringify('catppuccin-mocha') + ';',
	'',
	'export const themes: SiteTheme[] = ' +
		JSON.stringify(
			themes.map((t) => ({
				id: t.id,
				label: t.label,
				dot: { bg: hex(t.palette.mantle), fg: hex(t.palette.text), accent: hex(t.palette.primary) },
			})),
			null,
			'\t',
		),
	';',
	'',
].join('\n');

fs.mkdirSync(path.dirname(outCss), { recursive: true });
fs.writeFileSync(outCss, `export const themeCss = ${JSON.stringify(css)};\n`);
fs.writeFileSync(outTs, ts);

const lifted = themes.flatMap((t) => Object.entries(t.fixes).map(([k, v]) => `${t.id}.${k} ${(v * 100).toFixed(0)}%`));
console.log(`${themes.length} themes -> ${path.relative(process.cwd(), outCss)} + themes.ts`);
console.log(lifted.length ? `contrast lifts: ${lifted.join(', ')}` : 'contrast lifts: none');
