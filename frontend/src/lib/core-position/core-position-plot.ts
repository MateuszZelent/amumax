import { get } from 'svelte/store';
import { tablePlotState } from '$api/incoming/table-plot';
import {
	ECHARTS_THEME_NAME,
	THEME,
	ensureAmumaxEChartsTheme,
	initECharts,
	type ECharts
} from '$lib/theme/echarts-theme';

let chartInstance: ECharts | undefined;
let resizeObserver: ResizeObserver | null = null;
let equalAspect = true;
let orbitWindowNs = 2; // configurable time window in nanoseconds

export interface CoreMetrics {
	radiusNm: number;
	centerXNm: number;
	centerYNm: number;
	frequencyGHz: number;
	totalPoints: number;
	orbitPoints: number;
	hasData: boolean;
}

let lastMetrics: CoreMetrics = {
	radiusNm: 0, centerXNm: 0, centerYNm: 0,
	frequencyGHz: 0, totalPoints: 0, orbitPoints: 0, hasData: false,
};

export function getMetrics(): CoreMetrics { return lastMetrics; }

export function setOrbitWindow(ns: number) {
	orbitWindowNs = Math.max(0.1, ns);
	if (chartInstance) updateCorePosition();
}

export function getOrbitWindow(): number { return orbitWindowNs; }

export function setEqualAspect(val: boolean) {
	equalAspect = val;
	if (chartInstance) updateCorePosition();
}

export function getEqualAspect(): boolean { return equalAspect; }

export function plotCorePosition() {
	if (chartInstance === undefined) {
		initCorePosition();
	}
	updateCorePosition();
}

// ── Analytics ───────────────────────────────────────────────

function estimateFrequency(x: number[], y: number[], t: number[]): number {
	// Count zero-crossings of the angle from center to estimate period
	if (x.length < 4) return 0;
	let cx = 0, cy = 0;
	for (let i = 0; i < x.length; i++) { cx += x[i]; cy += y[i]; }
	cx /= x.length; cy /= y.length;

	// Compute angle and count complete revolutions
	let revolutions = 0;
	let prevAngle = Math.atan2(y[0] - cy, x[0] - cx);
	let totalAngle = 0;
	for (let i = 1; i < x.length; i++) {
		let angle = Math.atan2(y[i] - cy, x[i] - cx);
		let delta = angle - prevAngle;
		// Unwrap
		if (delta > Math.PI) delta -= 2 * Math.PI;
		if (delta < -Math.PI) delta += 2 * Math.PI;
		totalAngle += delta;
		prevAngle = angle;
	}
	revolutions = Math.abs(totalAngle) / (2 * Math.PI);
	const dt = t[t.length - 1] - t[0];
	if (dt <= 0) return 0;
	return revolutions / dt / 1e9; // GHz
}

// ── Chart update ────────────────────────────────────────────

function updateCorePosition() {
	if (!chartInstance) return;

	const state = get(tablePlotState);
	const rows = state.corePos;
	if (!rows || rows.length === 0) {
		chartInstance.setOption({ series: [] });
		lastMetrics = { ...lastMetrics, hasData: false };
		return;
	}

	const tArr = rows.map((r) => r[0]);
	const xArr = rows.map((r) => r[1]);
	const yArr = rows.map((r) => r[2]);

	// Split by time window
	const tMax = tArr[tArr.length - 1];
	const tCut = tMax - orbitWindowNs * 1e-9;
	let splitIdx = 0;
	for (let i = tArr.length - 1; i >= 0; i--) {
		if (tArr[i] < tCut) { splitIdx = i + 1; break; }
	}

	const oldData: number[][] = [];
	for (let i = 0; i < splitIdx; i++) oldData.push([xArr[i], yArr[i]]);

	const recentX = xArr.slice(splitIdx);
	const recentY = yArr.slice(splitIdx);
	const recentT = tArr.slice(splitIdx);
	const recentData = recentX.map((x, i) => [x, recentY[i]]);

	// Center & radius of orbit window
	const n = recentX.length || 1;
	let cx = 0, cy = 0;
	for (let i = 0; i < recentX.length; i++) { cx += recentX[i]; cy += recentY[i]; }
	cx /= n; cy /= n;

	let avgRadius = 0;
	for (let i = 0; i < recentX.length; i++) {
		avgRadius += Math.sqrt((recentX[i] - cx) ** 2 + (recentY[i] - cy) ** 2);
	}
	avgRadius /= n;

	// Frequency estimate
	const freq = estimateFrequency(recentX, recentY, recentT);

	// Update metrics
	lastMetrics = {
		radiusNm: avgRadius * 1e9,
		centerXNm: cx * 1e9,
		centerYNm: cy * 1e9,
		frequencyGHz: freq,
		totalPoints: rows.length,
		orbitPoints: recentX.length,
		hasData: true,
	};

	// Reference circle
	const circleData: number[][] = [];
	for (let i = 0; i <= 128; i++) {
		const th = (2 * Math.PI * i) / 128;
		circleData.push([cx + avgRadius * Math.cos(th), cy + avgRadius * Math.sin(th)]);
	}

	// Center crosshair
	const crossSize = avgRadius * 0.3 || 1e-10;
	const crossH = [[cx - crossSize, cy], [cx + crossSize, cy]];
	const crossV = [[cx, cy - crossSize], [cx, cy + crossSize]];

	// Axis ranges
	let xmin = Math.min(...xArr), xmax = Math.max(...xArr);
	let ymin = Math.min(...yArr), ymax = Math.max(...yArr);
	const pad = Math.max((xmax - xmin), (ymax - ymin)) * 0.12 || avgRadius || 1e-9;
	xmin -= pad; xmax += pad; ymin -= pad; ymax += pad;

	if (equalAspect) {
		const range = Math.max(xmax - xmin, ymax - ymin);
		const mx = (xmin + xmax) / 2, my = (ymin + ymax) / 2;
		xmin = mx - range / 2; xmax = mx + range / 2;
		ymin = my - range / 2; ymax = my + range / 2;
	}

	chartInstance.setOption({
		xAxis: { min: xmin, max: xmax },
		yAxis: { min: ymin, max: ymax },
		series: [
			{
				name: 'History',
				type: 'scatter',
				data: oldData,
				symbolSize: 1.5,
				itemStyle: { color: 'rgba(96, 165, 250, 0.15)' },
				z: 1,
			},
			{
				name: `Last ${orbitWindowNs} ns`,
				type: 'scatter',
				data: recentData,
				symbolSize: 2.5,
				itemStyle: { color: THEME.accent },
				z: 2,
			},
			{
				name: 'Avg orbit',
				type: 'line',
				data: circleData,
				showSymbol: false,
				lineStyle: { color: '#34d399', width: 1.5, type: 'dashed' },
				itemStyle: { color: '#34d399' },
				z: 3,
			},
			{
				name: 'Center',
				type: 'line',
				data: [...crossH, [null, null], ...crossV],
				showSymbol: false,
				lineStyle: { color: '#f59e0b', width: 1.5 },
				itemStyle: { color: '#f59e0b' },
				z: 4,
			},
		],
	});
}

// ── Init ────────────────────────────────────────────────────

function initCorePosition() {
	const chartDom = document.getElementById('core-position-plot');
	if (!chartDom) {
		setTimeout(() => { initCorePosition(); updateCorePosition(); }, 100);
		return;
	}

	ensureAmumaxEChartsTheme();
	chartInstance = initECharts(chartDom, ECHARTS_THEME_NAME, { renderer: 'canvas', useDirtyRect: true });

	chartInstance.setOption({
		animation: false,
		legend: { show: false },
		tooltip: {
			trigger: 'item',
			backgroundColor: THEME.tooltipBg,
			borderColor: THEME.tooltipBorder,
			textStyle: { color: THEME.tooltipText, fontSize: 11 },
			formatter: (p: any) => {
				if (!p.data || p.data.length < 2) return '';
				return `x: ${(p.data[0] * 1e9).toFixed(3)} nm<br/>y: ${(p.data[1] * 1e9).toFixed(3)} nm`;
			},
		},
		grid: {
			containLabel: true,
			left: 12, right: 12, top: 12, bottom: 12,
		},
		xAxis: {
			name: 'x (m)',
			nameLocation: 'middle',
			nameGap: 22,
			nameTextStyle: { color: THEME.text3, fontSize: 10 },
			axisTick: { show: false },
			axisLine: { lineStyle: { color: THEME.border } },
			splitLine: { lineStyle: { color: 'rgba(30, 45, 74, 0.4)' } },
			axisLabel: {
				formatter: (v: string) => (parseFloat(v) * 1e9).toFixed(1),
				color: THEME.text3,
				fontSize: 10,
			},
		},
		yAxis: {
			name: 'y (m)',
			nameLocation: 'middle',
			nameGap: 40,
			nameTextStyle: { color: THEME.text3, fontSize: 10 },
			axisTick: { show: false },
			axisLine: { lineStyle: { color: THEME.border } },
			splitLine: { lineStyle: { color: 'rgba(30, 45, 74, 0.4)' } },
			axisLabel: {
				formatter: (v: string) => (parseFloat(v) * 1e9).toFixed(1),
				color: THEME.text3,
				fontSize: 10,
			},
		},
		series: [],
	});

	if (!resizeObserver) {
		resizeObserver = new ResizeObserver(() => {
			if (chartInstance && !chartInstance.isDisposed()) chartInstance.resize();
		});
	}
	resizeObserver.disconnect();
	resizeObserver.observe(chartDom);
	chartInstance.resize();
}
