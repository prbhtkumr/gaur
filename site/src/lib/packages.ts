export interface Package {
	source: string;
	name: string;
	version: string;
	desc: string;
}

export const allPackages: Package[] = [
	{ source: 'extra', name: 'firefox', version: '133.0.3-1', desc: 'Fast, Private & Safe Web Browser' },
	  { source: 'extra', name: 'firefox-i18n-af', version: '133.0.3-1', desc: 'Afrikaans language pack for Firefox' },
	  { source: 'extra', name: 'firefox-i18n-ar', version: '133.0.3-1', desc: 'Arabic language pack for Firefox' },
	  { source: 'extra', name: 'firefox-i18n-be', version: '133.0.3-1', desc: 'Belarusian language pack for Firefox' },
	  { source: 'extra', name: 'firefox-developer-edition', version: '134.0b1-1', desc: 'Developer Edition of Firefox' },
	  { source: 'extra', name: 'firefox-nightly-bin', version: '135.0a1-1', desc: 'Nightly build of Firefox' },
	  { source: 'aur', name: 'firefoxpwa', version: '2.18.2-1', desc: 'A system to use Firefox as a PWA' },
	  { source: 'aur', name: 'zen-browser-bin', version: '1.0.0-a.42-1', desc: 'Experience tranquility while browsing the web without losing speed. Firefox based browser with vertical tabs.' },
	  { source: 'aur', name: 'zen-browser-avx2-bin', version: '1.0.0-a.42-1', desc: 'Zen Browser (AVX2 optimized) - precompiled binaries' },
	  { source: 'aur', name: 'zen-browser-patched-bin', version: '1.0.0-1', desc: 'Patched version of zen browser' },
	  { source: 'aur', name: 'zenity-git', version: '3.99-1', desc: 'Display GTK+ dialogs' },
	  { source: 'aur', name: 'zengarden', version: '0.1-1', desc: 'Zen garden theme' },
	  { source: 'core', name: 'linux', version: '6.12.1.arch1-1', desc: 'The Linux kernel and modules' },
	  { source: 'core', name: 'linux-headers', version: '6.12.1.arch1-1', desc: 'Headers and scripts for building modules for the Mainline kernel' },
	  { source: 'core', name: 'linux-lts', version: '6.6.63-1', desc: 'The Linux-LTS kernel and modules' },
	  { source: 'extra', name: 'linux-firmware', version: '20241118.9d3568b-1', desc: 'Firmware files for Linux' },
	  { source: 'aur', name: 'linux-zen', version: '6.12.1.zen1-1', desc: 'The Linux-ZEN kernel and modules' },
	  { source: 'aur', name: 'linux-mainline', version: '6.13rc1-1', desc: 'The Mainline Linux kernel and modules' },
	  { source: 'multilib', name: 'flash-lib32', version: '32.0-1', desc: 'Flash library' },
	  { source: 'core', name: 'filesystem', version: '2024.11.22-1', desc: 'Base Arch Linux files' }
];
