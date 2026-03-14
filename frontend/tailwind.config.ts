import type { Config } from 'tailwindcss';

export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	darkMode: 'class',
	theme: {
		extend: {
			colors: {
				bg: 'var(--bg)',
				'surface-1': 'var(--surface-1)',
				'surface-2': 'var(--surface-2)',
				'surface-3': 'var(--surface-3)',
				border: 'var(--border)',
				'border-subtle': 'var(--border-subtle)',
				'border-int': 'var(--border-interactive)',
				'text-1': 'var(--text-1)',
				'text-2': 'var(--text-2)',
				'text-3': 'var(--text-3)',
				accent: 'var(--accent)',
				'accent-hover': 'var(--accent-hover)',
				info: 'var(--info)',
				warn: 'var(--warn)',
				danger: 'var(--danger)',
				success: 'var(--success)'
			},
			fontFamily: {
				ui: ['IBM Plex Sans', '-apple-system', 'BlinkMacSystemFont', 'sans-serif'],
				mono: ['IBM Plex Mono', 'Consolas', 'Fira Code', 'monospace']
			},
			borderRadius: {
				sm: 'var(--radius-sm)',
				md: 'var(--radius-md)',
				lg: 'var(--radius-lg)'
			},
			boxShadow: {
				panel: 'var(--shadow-panel)',
				soft: 'var(--shadow-soft)',
				focus: 'var(--focus-ring)'
			},
			zIndex: {
				sticky: '10',
				overlay: '100',
				popout: '1000',
				modal: '2000',
				toast: '3000'
			}
		}
	}
} as Config;
