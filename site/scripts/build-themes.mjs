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

const derive = (t) => {
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

	const rules = {
		text: { shells: [base, mantle, crust, surface0, row], needs: [[1, 7], [0.9, 4.5], [0.7, 4.5]] },
		muted: { shells: [base, mantle, crust, card], needs: [[1, 4.8], [0.8, 4.5]] },
		overlay2: { shells: [base, row, surface0, crust], needs: [[1, 4.5]] },
		mauve: { shells: [base, mantle, crust, card], needs: [[1, 5], [0.8, 4.5]] },
		green: { shells: [crust, base, row], needs: [[1, 7], [0.9, 4.5]] },
		highlight: { shells: [base, mantle, card, surface0], needs: [[1, 4.5]] },
		pink: { shells: [surface0, base, row], needs: [[1, 4.5]] },
		blue: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
		yellow: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
		peach: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
		sapphire: { shells: [crust, base, row, surface0], needs: [[1, 4.5]] },
	};

	const fixes = {};
	const sources = {
		text: rawText,
		muted: mutedRaw,
		overlay2: overlayRaw,
		mauve: t.selected,
		green: t.success,
		highlight: t.dashboard_value,
		pink: t.dashboard,
		blue: t.install,
		yellow: t.warning,
		peach: t.multilib,
		sapphire: t.accent,
	};

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
		if (best.lift > 0) fixes[key] = best.lift;
		palette[key] = best.color;
	}
	return { palette, fixes };
};

const ALIASES = {
	background: 'base',
	surface: 'mantle',
	subtext0: 'muted',
	primary: 'mauve',
};

const files = fs.readdirSync(themesDir).filter((f) => f.endsWith('.toml')).sort();
const themes = files.map((f) => {
	const id = f.replace(/\.toml$/, '').replace(/[_-]/g, '-');
	const toml = parseToml(fs.readFileSync(path.join(themesDir, f), 'utf8'));
	for (const key of ['scrollbar_track', 'text', 'selected', 'success', 'dashboard_value', 'dashboard', 'install', 'warning', 'multilib', 'accent']) {
		if (!toml[key]) throw new Error(`${f}: missing "${key}"`);
	}
	const { palette, fixes } = derive(toml);
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
				dot: { bg: hex(t.palette.mantle), fg: hex(t.palette.text), accent: hex(t.palette.mauve) },
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
