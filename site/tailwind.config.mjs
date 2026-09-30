/** `rgb(var(--c-*) / <alpha-value>)` keeps Tailwind's opacity modifiers working
    against the theme custom properties emitted by scripts/build-themes.mjs. */
const themed = (name) => `rgb(var(--c-${name}) / <alpha-value>)`;

/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
	theme: {
		extend: {
			colors: {
				// Palette names, driven by gaur's own theme TOMLs
				mauve: themed('mauve'),
				pink: themed('pink'),
				blue: themed('blue'),
				yellow: themed('yellow'),
				green: themed('green'),
				peach: themed('peach'),
				sapphire: themed('sapphire'),
				text: themed('text'),
				subtext0: themed('muted'),
				overlay2: themed('overlay2'),
				surface0: themed('surface0'),
				surface1: themed('surface1'),
				surface2: themed('surface2'),
				base: themed('base'),
				mantle: themed('mantle'),
				crust: themed('crust'),

				// Semantic aliases — `primary` is the per-theme site accent
				// resolved by scripts/build-themes.mjs
				primary: themed('primary'),
				background: themed('base'),
				surface: themed('mantle'),
				muted: themed('muted'),
				accent: themed('green'),
				highlight: themed('highlight'),

				// Not present in the theme TOMLs; never referenced by a class
				rosewater: '#f5e0dc',
				flamingo: '#f2cdcd',
				red: '#f38ba8',
				maroon: '#eba0ac',
				teal: '#94e2d5',
				sky: '#89dceb',
				lavender: '#b4befe',
				subtext1: '#bac2de',
				overlay1: '#7f849c',
				overlay0: '#6c7086',
			},
			fontFamily: {
				mono: [
					'"CaskaydiaCove Nerd Font"',
					'"CaskaydiaCove Nerd Font Mono"',
					'"CaskaydiaCove NF"',
					'"JetBrains Mono"',
					'"Cascadia Code"',
					'"Fira Code"',
					'"DejaVu Sans Mono"',
					'"Liberation Mono"',
					'menlo',
					'monaco',
					'consolas',
					'"Courier New"',
					'monospace',
				],
			},

			borderRadius: {
				gaur: '8px',
			},
			boxShadow: {
				glow: '0 0 15px rgba(var(--c-highlight) / 0.2)',
			}
		},
	},
	plugins: [],
};
