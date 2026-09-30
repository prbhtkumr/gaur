/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
	theme: {
		extend: {
			colors: {
				// Catppuccin Mocha Palette
				rosewater: '#f5e0dc',
				flamingo: '#f2cdcd',
				pink: '#f5c2e7',
				mauve: '#cba6f7',
				red: '#f38ba8',
				maroon: '#eba0ac',
				peach: '#fab387',
				yellow: '#f9e2af',
				green: '#a6e3a1',
				teal: '#94e2d5',
				sky: '#89dceb',
				sapphire: '#74c7ec',
				blue: '#89b4fa',
				lavender: '#b4befe',
				text: '#cdd6f4',
				subtext1: '#bac2de',
				subtext0: '#a6adc8',
				overlay2: '#949cbb',
				overlay1: '#7f849c',
				overlay0: '#6c7086',
				surface2: '#585b70',
				surface1: '#45475a',
				surface0: '#313244',
				base: '#1e1e2e',
				mantle: '#181825',
				crust: '#11111b',

				// Semantic Aliases
				primary: '#cba6f7',
				background: '#1e1e2e',
				surface: '#181825',
				muted: '#a6adc8',
				accent: '#a6e3a1',
				highlight: '#74c7ec',
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
				glow: '0 0 15px rgba(116, 199, 236, 0.2)',
			}
		},
	},
	plugins: [],
};
