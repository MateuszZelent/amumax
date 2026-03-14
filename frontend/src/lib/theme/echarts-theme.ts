/**
 * Shared ECharts theme constants for the MicroLab design system.
 * Both preview2D.ts and table-plot.ts import from here
 * instead of hardcoding color values.
 */

// ─── MicroLab palette for ECharts (CSS vars not accessible in JS canvas) ─────
export const THEME = {
	// Surfaces
	bg:              '#080d1a',
	surface1:        '#0f172a',
	surface2:        '#1a2338',
	surface3:        '#1e293b',

	// Borders
	border:          '#1e2d4a',
	borderInteractive: '#2a3f66',

	// Text
	text1:           '#e2e8f0',
	text2:           '#94a3b8',
	text3:           '#5a6b8a',

	// Accent
	accent:          '#3b82f6',
	accentHover:     '#2563eb',
	info:            '#60a5fa',

	// Tooltip
	tooltipBg:       '#0f172a',
	tooltipBorder:   '#3b82f6',
	tooltipText:     '#e2e8f0',

	// Toolbox
	toolboxIcon:     '#94a3b8',

	// Brush
	brushBg:         'rgba(59, 130, 246, 0.15)',
	brushBorder:     '#3b82f6',
} as const;
