import { use, init, registerTheme, type ECharts } from 'echarts/core';
import { HeatmapChart, LineChart } from 'echarts/charts';
import {
	AxisPointerComponent,
	DataZoomInsideComponent,
	DataZoomSliderComponent,
	GridComponent,
	ToolboxComponent,
	TooltipComponent,
	VisualMapComponent
} from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

use([
	LineChart,
	HeatmapChart,
	GridComponent,
	TooltipComponent,
	ToolboxComponent,
	VisualMapComponent,
	AxisPointerComponent,
	DataZoomInsideComponent,
	DataZoomSliderComponent,
	CanvasRenderer
]);

export { init as initECharts, type ECharts };

export const THEME = {
	bg: '#080d1a',
	surface1: '#0f172a',
	surface2: '#1a2338',
	surface3: '#1e293b',
	border: '#1e2d4a',
	borderInteractive: '#2a3f66',
	text1: '#e2e8f0',
	text2: '#94a3b8',
	text3: '#5a6b8a',
	accent: '#3b82f6',
	accentHover: '#2563eb',
	info: '#60a5fa',
	tooltipBg: '#0f172a',
	tooltipBorder: '#3b82f6',
	tooltipText: '#e2e8f0',
	toolboxIcon: '#94a3b8',
	brushBg: 'rgba(59, 130, 246, 0.15)',
	brushBorder: '#3b82f6'
} as const;

export const ECHARTS_THEME_NAME = 'amumax-lab';

let registered = false;

export function ensureAmumaxEChartsTheme() {
	if (registered) {
		return;
	}

	registerTheme(ECHARTS_THEME_NAME, {
		color: [THEME.info, THEME.accent, '#7dd3fc', '#34d399'],
		backgroundColor: 'transparent',
		textStyle: {
			color: THEME.text2,
			fontFamily: 'IBM Plex Sans, sans-serif'
		}
	});

	registered = true;
}
