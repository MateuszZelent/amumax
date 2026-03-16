import { get } from 'svelte/store';
import { fftState } from '$api/incoming/fft';
import {
    ECHARTS_THEME_NAME,
    THEME,
    ensureAmumaxEChartsTheme,
    initECharts,
    type ECharts
} from '$lib/theme/echarts-theme';

let spectrumChart: ECharts | undefined;
let spectrogramChart: ECharts | undefined;
let spectrumResizeObserver: ResizeObserver | null = null;
let spectrogramResizeObserver: ResizeObserver | null = null;

// --- Spectrum ---

export function plotSpectrum() {
    if (spectrumChart === undefined) {
        initSpectrum();
    }
    updateSpectrum();
}

function initSpectrum() {
    const dom = document.getElementById('fft-spectrum');
    if (!dom) {
        setTimeout(initSpectrum, 100);
        return;
    }
    ensureAmumaxEChartsTheme();
    spectrumChart = initECharts(dom, ECHARTS_THEME_NAME, { renderer: 'canvas', useDirtyRect: true });

    // Distinct colors from app palette: accent(teal), warn(amber), danger(coral), info(blue), success(green)
    const SERIES_COLORS = ['#57c8b6', '#f2b45a', '#ff7c7c', '#6ba7ff', '#68d39a', '#c084fc'];

    const f = get(fftState);
    const series = (f.labels || []).map((label: string, i: number) => ({
        type: 'line' as const,
        name: label,
        showSymbol: false,
        sampling: 'lttb' as const,
        animation: false,
        data: [] as number[][],
        lineStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length], width: 2 },
        itemStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length] },
    }));

    spectrumChart.setOption({
        animation: false,
        legend: {
            show: true,
            textStyle: { color: THEME.text2 },
            top: 0,
        },
        tooltip: {
            trigger: 'axis',
            backgroundColor: THEME.tooltipBg,
            textStyle: { color: THEME.tooltipText },
        },
        grid: {
            containLabel: false,
            left: '10%',
            right: '5%',
            top: '12%',
            bottom: '15%',
        },
        xAxis: {
            name: 'Frequency (GHz)',
            nameLocation: 'middle',
            nameGap: 25,
            nameTextStyle: { color: THEME.text2 },
            axisTick: { lineStyle: { color: THEME.border } },
            axisLabel: {
                color: THEME.text2,
                formatter: (v: number) => v.toFixed(1),
            },
        },
        yAxis: {
            name: '|FFT|',
            nameLocation: 'middle',
            nameGap: 45,
            nameTextStyle: { color: THEME.text2 },
            axisTick: { lineStyle: { color: THEME.border } },
            axisLabel: {
                color: THEME.text2,
                formatter: (v: number) => v.toPrecision(2),
            },
        },
        series,
    });

    if (!spectrumResizeObserver) {
        spectrumResizeObserver = new ResizeObserver(() => {
            if (spectrumChart && !spectrumChart.isDisposed()) spectrumChart.resize();
        });
    }
    spectrumResizeObserver.disconnect();
    spectrumResizeObserver.observe(dom);
}

function updateSpectrum() {
    if (!spectrumChart) return;
    const f = get(fftState);
    if (!f.freqAxis || !f.spectrum || f.spectrum.length === 0) return;

    const SERIES_COLORS = ['#57c8b6', '#f2b45a', '#ff7c7c', '#6ba7ff', '#68d39a', '#c084fc'];

    const series = f.spectrum.map((magnitudes: number[], i: number) => ({
        name: f.labels?.[i] || `comp${i}`,
        data: magnitudes.map((mag: number, fi: number) => [f.freqAxis[fi], mag]),
        lineStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length], width: 2 },
        itemStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length] },
    }));

    spectrumChart.setOption({ series });
}

// --- Spectrogram ---

export function plotSpectrogram() {
    if (spectrogramChart === undefined) {
        initSpectrogram();
    }
    updateSpectrogram();
}

function initSpectrogram() {
    const dom = document.getElementById('fft-spectrogram');
    if (!dom) {
        setTimeout(initSpectrogram, 100);
        return;
    }
    ensureAmumaxEChartsTheme();
    spectrogramChart = initECharts(dom, ECHARTS_THEME_NAME, { renderer: 'canvas', useDirtyRect: true });

    spectrogramChart.setOption({
        animation: false,
        tooltip: {
            position: 'top',
            backgroundColor: THEME.tooltipBg,
            textStyle: { color: THEME.tooltipText },
        },
        grid: {
            containLabel: false,
            left: '10%',
            right: '8%',
            top: '5%',
            bottom: '15%',
        },
        xAxis: {
            name: 'Frequency (GHz)',
            nameLocation: 'middle',
            nameGap: 25,
            nameTextStyle: { color: THEME.text2 },
            type: 'category',
            data: [],
            axisLabel: {
                color: THEME.text2,
                interval: 'auto',
            },
        },
        yAxis: {
            name: 'Time (ns)',
            nameLocation: 'middle',
            nameGap: 45,
            nameTextStyle: { color: THEME.text2 },
            type: 'category',
            data: [],
            axisLabel: {
                color: THEME.text2,
                interval: 'auto',
            },
        },
        visualMap: {
            min: 0,
            max: 1,
            calculable: true,
            orient: 'vertical',
            right: 0,
            top: '5%',
            bottom: '15%',
            textStyle: { color: THEME.text2 },
            inRange: {
                color: ['#0d0829', '#1b0c5e', '#5b1493', '#9c1fa0', '#d54d88', '#f48c50', '#fcdc76', '#fcffc9'],
            },
        },
        series: [{
            type: 'heatmap',
            data: [],
            emphasis: {
                itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0, 0, 0, 0.5)' },
            },
        }],
    });

    if (!spectrogramResizeObserver) {
        spectrogramResizeObserver = new ResizeObserver(() => {
            if (spectrogramChart && !spectrogramChart.isDisposed()) spectrogramChart.resize();
        });
    }
    spectrogramResizeObserver.disconnect();
    spectrogramResizeObserver.observe(dom);
}

function updateSpectrogram() {
    if (!spectrogramChart) return;
    const f = get(fftState);
    if (!f.spectrogram || f.spectrogram.length === 0 || !f.freqAxis) return;

    const freqLabels = f.freqAxis.map((freq: number) => freq.toFixed(1));
    const timeLabels = (f.spectrogramTimes || []).map((t: number) => (t * 1e9).toFixed(2));

    // Build heatmap data: [freqIndex, timeIndex, value]
    const heatData: number[][] = [];
    let maxVal = 0;
    for (let ti = 0; ti < f.spectrogram.length; ti++) {
        for (let fi = 0; fi < f.spectrogram[ti].length; fi++) {
            const val = f.spectrogram[ti][fi];
            heatData.push([fi, ti, val]);
            if (val > maxVal) maxVal = val;
        }
    }

    spectrogramChart.setOption({
        xAxis: { data: freqLabels },
        yAxis: { data: timeLabels },
        visualMap: { max: maxVal || 1 },
        series: [{ data: heatData }],
    });
}
