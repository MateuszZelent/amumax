import { previewState } from '$api/incoming/preview';
import { get } from 'svelte/store';
import { meshState } from '$api/incoming/mesh';
import { disposePreview3D } from './preview3D';
import {
	ECHARTS_THEME_NAME,
	THEME,
	ensureAmumaxEChartsTheme,
	initECharts,
	type ECharts
} from '$lib/theme/echarts-theme';

let chartInstance: ECharts | undefined;
let resizeObserver: ResizeObserver | null = null;

type ColorScale = {
	min: number;
	max: number;
	palette: string[];
};

type AxisMetrics = {
	xExtentNm: number;
	yExtentNm: number;
	xDisplayStepNm: number;
	yDisplayStepNm: number;
	xCategories: number[];
	yCategories: number[];
};

export function preview2D() {
	const state = get(previewState);
	if (!state.scalarField || state.scalarField.length === 0) {
		disposePreview2D();
		disposePreview3D();
		return;
	}

	// Dispose 3D renderer if it was active
	disposePreview3D();

	const container = document.getElementById('container');
	if (!container) {
		return;
	}

	// Create chart instance only when truly needed (first time or after explicit dispose)
	if (chartInstance === undefined || chartInstance.isDisposed()) {
		init();
		return;
	}

	// Keep updates incremental to avoid visible canvas resets/flicker.
	updateData();
}

function getColorScale(min: number, max: number): ColorScale {
	if (min < 0 && max > 0) {
		const bound = Math.max(Math.abs(min), Math.abs(max));
		return {
			min: -bound,
			max: bound,
			palette: ['#15315f', '#2f6caa', '#90b9df', '#f4f1ed', '#efb09d', '#cf6256', '#7d1d34']
		};
	}

	if (max <= 0) {
		return {
			min,
			max,
			palette: ['#f3f7fd', '#cfdef1', '#91b8dd', '#5688bd', '#285b93', '#14365f']
		};
	}

	return {
		min,
		max,
		palette: ['#0a1220', '#143d67', '#1c6d8f', '#24a0a4', '#8ed6ac', '#f1f7bb']
	};
}

function formatMagnitude(value: number) {
	if (!Number.isFinite(value)) {
		return 'NaN';
	}

	if (value === 0) {
		return '0';
	}

	const abs = Math.abs(value);
	if (abs >= 1000 || abs < 1e-2) {
		return value.toExponential(2);
	}
	if (abs >= 10) {
		return value.toFixed(1);
	}
	if (abs >= 1) {
		return value.toFixed(2);
	}
	return value.toPrecision(2);
}

function formatDistanceNm(distanceNm: number) {
	if (!Number.isFinite(distanceNm)) {
		return 'NaN';
	}

	const abs = Math.abs(distanceNm);
	if (abs >= 1000) {
		return distanceNm.toFixed(0);
	}

	if (abs >= 100) {
		return distanceNm.toFixed(0);
	}

	if (abs >= 10) {
		return distanceNm.toFixed(1);
	}

	return distanceNm.toFixed(2);
}

function axisPointerLabelFormatter(axis: 'x' | 'y') {
	return function (params: { value?: number }) {
		if (params.value === undefined) {
			return 'NaN';
		}
		return `${axis}: ${formatDistanceNm(Number(params.value))} nm`;
	};
}

function buildVisualMap(quantity: string, unit: string, min: number, max: number) {
	const scale = getColorScale(min, max);
	const unitSuffix = unit ? ` ${unit}` : '';

	return {
		type: 'continuous' as const,
		min: scale.min,
		max: scale.max,
		calculable: false,
		realtime: false,
		precision: 3,
		orient: 'vertical' as const,
		right: 8,
		top: 'middle' as const,
		itemWidth: 12,
		itemHeight: 188,
		align: 'right' as const,
		padding: [12, 10, 12, 10],
		textGap: 10,
		backgroundColor: 'rgba(15, 23, 42, 0.76)',
		borderColor: THEME.border,
		borderWidth: 1,
		text: [`${formatMagnitude(scale.max)}${unitSuffix}`, `${formatMagnitude(scale.min)}`],
		textStyle: {
			color: THEME.text2,
			fontSize: 11,
			fontWeight: 600
		},
		formatter: (value: number) => `${formatMagnitude(value)}${unitSuffix}`,
		inRange: {
			color: scale.palette
		},
		outOfRange: {
			color: ['rgba(107, 122, 154, 0.18)']
		},
		seriesIndex: 0,
		showLabel: true
	};
}

function getAxisMetrics(): AxisMetrics {
	const ps = get(previewState);
	const mesh = get(meshState);
	const xChosenSize = Math.max(ps.xChosenSize, 1);
	const yChosenSize = Math.max(ps.yChosenSize, 1);
	const xExtentNm = mesh.dx * 1e9 * mesh.Nx;
	const yExtentNm = mesh.dy * 1e9 * mesh.Ny;

	return {
		xExtentNm,
		yExtentNm,
		xDisplayStepNm: xChosenSize > 1 ? xExtentNm / (xChosenSize - 1) : xExtentNm,
		yDisplayStepNm: yChosenSize > 1 ? yExtentNm / (yChosenSize - 1) : yExtentNm,
		xCategories: Array.from({ length: xChosenSize }, (_, index) => index),
		yCategories: Array.from({ length: yChosenSize }, (_, index) => index)
	};
}

function tooltipFormatter(params: any) {
	const ps = get(previewState);
	if (params.value === undefined) {
		return 'NaN';
	}
	const { xDisplayStepNm, yDisplayStepNm } = getAxisMetrics();
	const xnm = Number(params.value[0]) * xDisplayStepNm;
	const ynm = Number(params.value[1]) * yDisplayStepNm;
	const value = Number(params.value[2]);
	const unitSuffix = ps.unit ? ` ${ps.unit}` : '';
	return [
		`<strong>${ps.quantity}</strong>`,
		`x: ${formatDistanceNm(xnm)} nm`,
		`y: ${formatDistanceNm(ynm)} nm`,
		`value: ${formatMagnitude(value)}${unitSuffix}`
	].join('<br/>');
}

/** Incremental update — only series data + axis bounds + visualMap range. */
function updateData() {
	if (chartInstance === undefined || chartInstance.isDisposed()) {
		init();
		return;
	}
	const ps = get(previewState);
	const { xCategories, yCategories, xDisplayStepNm, yDisplayStepNm } = getAxisMetrics();
	const visualMap = buildVisualMap(ps.quantity, ps.unit, ps.min, ps.max) as any;
	chartInstance.setOption(
		{
			animation: false,
			animationDurationUpdate: 0,
			xAxis: {
				data: xCategories,
				axisLabel: {
					formatter: function (value: number) {
						return formatDistanceNm(Number(value) * xDisplayStepNm);
					},
					hideOverlap: true
				}
			},
			yAxis: {
				data: yCategories,
				axisLabel: {
					formatter: function (value: number) {
						return formatDistanceNm(Number(value) * yDisplayStepNm);
					},
					hideOverlap: true
				}
			},
			series: [
				{
					name: ps.quantity,
					animation: false,
					progressive: 0,
					progressiveThreshold: Number.MAX_SAFE_INTEGER,
					data: ps.scalarField
				}
			],
			visualMap: [visualMap]
		},
		{ lazyUpdate: true }
	);
}

function init() {
	const chartDom = document.getElementById('container');
	if (!chartDom) {
		return;
	}
	// Reuse existing instance if possible — avoids canvas teardown/flicker.
	if (!chartInstance || chartInstance.isDisposed()) {
		ensureAmumaxEChartsTheme();
		chartInstance = initECharts(chartDom, ECHARTS_THEME_NAME, { renderer: 'canvas', useDirtyRect: true });
	}
	resizeECharts();
	setFullOptions();
}

/** Replace all chart options on the existing instance (no canvas destruction). */
function setFullOptions() {
	if (!chartInstance || chartInstance.isDisposed()) {
		return;
	}
	const ps = get(previewState);
	const { xCategories, yCategories, xDisplayStepNm, yDisplayStepNm } = getAxisMetrics();
	const visualMap = buildVisualMap(ps.quantity, ps.unit, ps.min, ps.max);

	// @ts-ignore
	chartInstance.setOption(
		{
			tooltip: {
				position: 'top',
				confine: true,
				formatter: tooltipFormatter,
				backgroundColor: THEME.tooltipBg,
				borderColor: THEME.tooltipBorder,
				borderWidth: 1,
				padding: [10, 12],
				textStyle: {
					color: THEME.tooltipText,
					fontSize: 12
				}
			},
			xAxis: {
				type: 'category',
				data: xCategories,
				name: 'x (nm)',
				nameLocation: 'middle',
				nameGap: 30,
				nameTextStyle: {
					color: THEME.text2,
					fontWeight: 600
				},
				axisLine: {
					show: true,
					lineStyle: {
						color: THEME.border
					}
				},
				axisPointer: {
					show: true,
					label: {
						show: true,
						backgroundColor: THEME.tooltipBg,
						color: THEME.tooltipText,
						padding: [6, 8],
						borderColor: THEME.accent,
						borderWidth: 1,
						formatter: axisPointerLabelFormatter('x')
					},
					lineStyle: {
						color: THEME.accent,
						width: 1.5,
						type: 'dashed'
					}
				},
				axisTick: {
					length: 6,
					lineStyle: {
						type: 'solid',
						color: THEME.border
					}
				},
				axisLabel: {
					show: true,
					formatter: function (value: number) {
						return formatDistanceNm(Number(value) * xDisplayStepNm);
					},
					color: THEME.text2,
					showMinLabel: true,
					showMaxLabel: true,
					hideOverlap: true
				},
				splitLine: {
					show: false
				}
			},
			yAxis: {
				type: 'category',
				data: yCategories,
				name: 'y (nm)',
				nameLocation: 'middle',
				nameGap: 54,
				nameTextStyle: {
					color: THEME.text2,
					fontWeight: 600
				},
				axisLine: {
					show: true,
					lineStyle: {
						color: THEME.border
					}
				},
				axisPointer: {
					show: true,
					label: {
						show: true,
						backgroundColor: THEME.tooltipBg,
						color: THEME.tooltipText,
						padding: [6, 8],
						borderColor: THEME.accent,
						borderWidth: 1,
						formatter: axisPointerLabelFormatter('y')
					},
					lineStyle: {
						color: THEME.accent,
						width: 1.5,
						type: 'dashed'
					}
				},
				axisTick: {
					length: 6,
					lineStyle: {
						type: 'solid',
						color: THEME.border
					}
				},
				axisLabel: {
					show: true,
					formatter: function (value: number) {
						return formatDistanceNm(Number(value) * yDisplayStepNm);
					},
					color: THEME.text2,
					showMinLabel: true,
					showMaxLabel: true,
					hideOverlap: true
				},
				splitLine: {
					show: false
				}
			},
			visualMap: [visualMap],
				series: [
					{
						name: ps.quantity,
						type: 'heatmap',
						selectedMode: false,
						emphasis: { disabled: true },
						// Disable progressive chunks to avoid visible left-to-right repainting on each refresh.
						progressive: 0,
						progressiveThreshold: Number.MAX_SAFE_INTEGER,
						animation: false,
						data: ps.scalarField
					}
				],
			grid: {
				containLabel: true,
				left: 58,
				right: 92,
				top: 42,
				bottom: 52
			},
			toolbox: {
				show: true,
				top: 10,
				right: 10,
				itemSize: 20,
				itemGap: 12,
				iconStyle: {
					borderColor: THEME.toolboxIcon,
					borderWidth: 1.15
				},
				emphasis: {
					iconStyle: {
						borderColor: THEME.text1
					}
				},
				feature: {
					dataZoom: {
						xAxisIndex: 0,
						yAxisIndex: 0,
						brushStyle: {
							color: THEME.brushBg,
							borderColor: THEME.brushBorder,
							borderWidth: 2
						}
					},
					dataView: { show: false },
					restore: {
						show: true
					},
					saveAsImage: {
						type: 'png',
						name: 'preview'
					}
				}
			},
				animation: false,
				animationDurationUpdate: 0
			},
			{ notMerge: true }
	);
}

export function disposePreview2D() {
	if (chartInstance && !chartInstance.isDisposed()) {
		chartInstance.dispose();
	}
	chartInstance = undefined;

	if (resizeObserver) {
		resizeObserver.disconnect();
		resizeObserver = null;
	}
}

export function resizeECharts() {
	const container = document.getElementById('container');
	if (!container) {
		return;
	}

	if (!resizeObserver) {
		resizeObserver = new ResizeObserver(() => {
			if (chartInstance === undefined || chartInstance.isDisposed()) {
				return;
			}
			chartInstance.resize();
		});
	}

	resizeObserver.disconnect();
	resizeObserver.observe(container);

	if (chartInstance !== undefined && !chartInstance.isDisposed()) {
		chartInstance.resize();
	}
}
