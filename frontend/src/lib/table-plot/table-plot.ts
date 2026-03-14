import { get } from 'svelte/store';
import { tablePlotState } from '$api/incoming/table-plot';
import {
	ECHARTS_THEME_NAME,
	THEME,
	ensureAmumaxEChartsTheme,
	initECharts,
	type ECharts
} from '$lib/theme/echarts-theme';

export function plotTable() {
    if (chartInstance === undefined) {
        init();
        update();
    } else {
        update();
    }
}

let chartInstance: ECharts;
let resizeObserver: ResizeObserver | null = null;

function update() {
    if (chartInstance === undefined) {
        return;
    }
    let t = get(tablePlotState);
    chartInstance.setOption({
        series: [
            {
                name: 'y',
                data: t.data,
            }
        ],
        xAxis: {
            name: `${t.xColumn} (${t.xColumnUnit})`,
            min: t.xmin,
            max: t.xmax,
        },
        yAxis: {
            name: `${t.yColumn} (${t.yColumnUnit})`,
            min: t.ymin,
            max: t.ymax,
        }
    });
}

export function init() {
    var chartDom = document.getElementById('table-plot')!;
    if (chartDom === null) {
        setTimeout(init, 100);
        return;
    }
    // https://apache.github.io/echarts-handbook/en/best-practices/canvas-vs-svg
    ensureAmumaxEChartsTheme();
    chartInstance = initECharts(chartDom, ECHARTS_THEME_NAME, { renderer: 'canvas', useDirtyRect: true });
    let t = get(tablePlotState);

    // @ts-ignore
    chartInstance.setOption({
        axisPointer: {
            show: true,
            type: 'line',
            lineStyle: {
                color: THEME.accent,
                width: 2,
                type: 'dashed'
            },

            label: {
                backgroundColor: THEME.tooltipBg,
                color: THEME.tooltipText,
                formatter: function (params: any) {
                    return parseFloat(params.value).toPrecision(2);
                },
                padding: [8, 5, 8, 5],
                borderColor: THEME.accent,
                borderWidth: 1,
            }
        },
        animation: false,
        grid: {
            containLabel: false,
            left: '10%',
            right: '10%',
        },
        xAxis: {
            name: `${t.xColumn} (${t.xColumnUnit})`,
            min: t.xmin,
            max: t.xmax,
            nameLocation: 'middle',
            nameGap: 25,
            nameTextStyle: {
                color: THEME.text2,
            },
            axisTick: {
                alignWithLabel: true,
                length: 6,
                lineStyle: {
                    type: 'solid',
                    color: THEME.border,
                },
            },
            axisLabel: {
                show: true,
                formatter: function (value: string, _index: string) {
                    return parseFloat(value).toPrecision(2);
                },
                color: THEME.text2,
                // showMinLabel: true,
            }
        },
        yAxis: {
            name: `${t.yColumn} (${t.yColumnUnit})`,
            min: t.ymin,
            max: t.ymax,
            nameLocation: 'middle',
            nameGap: 45,
            nameTextStyle: {
                color: THEME.text2,
            },
            axisTick: {
                alignWithLabel: true,
                length: 6,
                lineStyle: {
                    type: 'solid',
                    color: THEME.border,
                },
            },
            axisLabel: {
                show: true,
                formatter: function (value: string, _index: string) {
                    return parseFloat(value).toPrecision(2);
                },
                color: THEME.text2,
                showMinLabel: true,
            }
        },
        series: [
            {
                type: 'line',
                name: 'y',
                showSymbol: false,
                sampling: 'lttb',
                progressive: 2000,
                progressiveThreshold: 3000,
                animation: false,
                data: t.data,
            }
        ]
    });

    resizeECharts();
}

export function resizeECharts() {
    const chartDom = document.getElementById('table-plot');
    if (!chartDom) {
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
    resizeObserver.observe(chartDom);

    if (chartInstance !== undefined && !chartInstance.isDisposed()) {
        chartInstance.resize();
    }
}
