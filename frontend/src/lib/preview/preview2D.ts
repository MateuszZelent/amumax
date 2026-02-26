import { previewState } from '$api/incoming/preview';
import * as echarts from 'echarts';
import { get } from 'svelte/store';
import { meshState } from '$api/incoming/mesh';
import { disposePreview3D } from './preview3D';

let chartInstance: echarts.ECharts | undefined;
let resizeListenerAttached = false;

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

function getColorMap(min: number, max: number) {
	if (min < 0 && max > 0) {
		return ['#313695', '#ffffff', '#a50026'];
	}
	return ['#ffffff', '#a50026'];
}

function getAxisScale() {
	const ps = get(previewState);
	const mesh = get(meshState);
	const xDen = Math.max(ps.xChosenSize, 1);
	const yDen = Math.max(ps.yChosenSize, 1);
	return {
		xScale: (mesh.dx * 1e9 * mesh.Nx) / xDen,
		yScale: (mesh.dy * 1e9 * mesh.Ny) / yDen
	};
}

function tooltipFormatter(params: any) {
	const ps = get(previewState);
	if (params.value === undefined) {
		return 'NaN';
	}
	const { xScale, yScale } = getAxisScale();
	const xnm = (Number(params.value[0]) * xScale).toFixed(1);
	const ynm = (Number(params.value[1]) * yScale).toFixed(1);
	const value = Number(params.value[2]).toPrecision(6);
	return `x: ${xnm} nm<br/>y: ${ynm} nm<br/>${value} ${ps.unit}`;
}

/** Incremental update — only series data + axis bounds + visualMap range. */
function updateData() {
	if (chartInstance === undefined || chartInstance.isDisposed()) {
		init();
		return;
	}
	const ps = get(previewState);
	const { xScale, yScale } = getAxisScale();
	chartInstance.setOption(
		{
			animation: false,
			animationDurationUpdate: 0,
			xAxis: {
				max: Math.max(ps.xChosenSize - 1, 0),
				axisLabel: {
					formatter: function (value: number) {
						return (value * xScale).toFixed(0);
					}
				}
			},
			yAxis: {
				max: Math.max(ps.yChosenSize - 1, 0),
				axisLabel: {
					formatter: function (value: number) {
						return (value * yScale).toFixed(0);
					}
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
			visualMap: [
				{
					max: ps.max,
					min: ps.min,
					text: [ps.unit, ''],
					inRange: {
						color: getColorMap(ps.min, ps.max)
					}
				}
			]
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
		chartInstance = echarts.init(chartDom, undefined, { renderer: 'canvas', useDirtyRect: true });
	}
	setFullOptions();
}

/** Replace all chart options on the existing instance (no canvas destruction). */
function setFullOptions() {
	if (!chartInstance || chartInstance.isDisposed()) {
		return;
	}
	const ps = get(previewState);
	const { xScale, yScale } = getAxisScale();

	// @ts-ignore
	chartInstance.setOption(
		{
			tooltip: {
				position: 'top',
				formatter: tooltipFormatter,
				backgroundColor: '#282a36',
				borderColor: '#6e9bcb',
				textStyle: {
					color: '#fff'
				}
			},
			axisPointer: {
				show: true,
				type: 'line',
				triggerEmphasis: false,
				lineStyle: {
					color: '#6e9bcb',
					width: 2,
					type: 'dashed'
				},
				label: {
					backgroundColor: '#282a36',
					color: '#fff',
					formatter: function (params: any) {
						if (params.value === undefined) {
							return 'NaN';
						}
						return ` ${Number(params.value).toFixed(0)} idx`;
					},
					padding: [8, 5, 8, 5],
					borderColor: '#6e9bcb',
					borderWidth: 1
				}
			},
			xAxis: {
				type: 'value',
				min: 0,
				max: Math.max(ps.xChosenSize - 1, 0),
				name: 'x (nm)',
				nameLocation: 'middle',
				nameGap: 25,
				nameTextStyle: {
					color: '#fff'
				},
				axisTick: {
					length: 6,
					lineStyle: {
						type: 'solid',
						color: '#fff'
					}
				},
				axisLabel: {
					show: true,
					formatter: function (value: number) {
						return (value * xScale).toFixed(0);
					},
					color: '#fff',
					showMinLabel: true
				}
			},
			yAxis: {
				type: 'value',
				min: 0,
				max: Math.max(ps.yChosenSize - 1, 0),
				name: 'y (nm)',
				nameLocation: 'middle',
				nameGap: 45,
				nameTextStyle: {
					color: '#fff'
				},
				axisTick: {
					length: 6,
					lineStyle: {
						type: 'solid',
						color: '#fff'
					}
				},
				axisLabel: {
					show: true,
					formatter: function (value: number) {
						return (value * yScale).toFixed(0);
					},
					color: '#fff',
					showMinLabel: true
				}
			},
			visualMap: [
				{
					type: 'continuous',
					min: ps.min,
					max: ps.max,
					calculable: true,
					realtime: false,
					formatter: function (value: any) {
						return Number(value).toPrecision(2);
					},
					itemWidth: 9,
					itemHeight: 140,
					text: [ps.unit, ''],
					textStyle: {
						color: '#fff'
					},
					inRange: {
						color: getColorMap(ps.min, ps.max)
					},
					left: 'right'
				}
			],
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
				left: '10%',
				right: '10%'
			},
			toolbox: {
				show: true,
				itemSize: 20,
				iconStyle: {
					borderColor: '#fff'
				},
				feature: {
					dataZoom: {
						xAxisIndex: 0,
						yAxisIndex: 0,
						brushStyle: {
							color: '#282a3655',
							borderColor: '#6e9bcb',
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
	const container = document.getElementById('container');
	if (container) {
		const echartsInstance = echarts.getInstanceByDom(container);
		if (echartsInstance) {
			echartsInstance.dispose();
		}
	}
	chartInstance = undefined;
}

export function resizeECharts() {
	if (resizeListenerAttached) {
		return;
	}
	window.addEventListener('resize', function () {
		if (chartInstance === undefined || chartInstance.isDisposed()) {
			return;
		}
		chartInstance.resize();
	});
	resizeListenerAttached = true;
}
