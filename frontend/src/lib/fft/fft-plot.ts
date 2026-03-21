import { get } from 'svelte/store';
import { fftState } from '$api/incoming/fft';
import {
    ECHARTS_THEME_NAME,
    THEME,
    ensureAmumaxEChartsTheme,
    initECharts,
    type ECharts
} from '$lib/theme/echarts-theme';
import { LegendComponent } from 'echarts/components';
import { use } from 'echarts/core';
use([LegendComponent]);

// Distinct colors from app palette
const SERIES_COLORS = ['#57c8b6', '#f2b45a', '#ff7c7c', '#6ba7ff', '#68d39a', '#c084fc'];

let spectrumChart: ECharts | undefined;
let spectrogramChart: ECharts | undefined;
let spectrumResizeObserver: ResizeObserver | null = null;
let spectrogramResizeObserver: ResizeObserver | null = null;

// --- Spectrum (items 1, 3) ---

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
            formatter: (params: Array<{ seriesName: string; data: number[]; color: string }>) => {
                if (!Array.isArray(params) || params.length === 0) return '';
                const freqGHz = params[0].data[0];
                let html = `<b>${freqGHz.toFixed(3)} GHz</b>`;
                for (const p of params) {
                    const amp = p.data[1];
                    const ampStr = amp < 0.001 ? amp.toExponential(3) : amp.toPrecision(4);
                    html += `<br/><span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${p.color};margin-right:5px;"></span>${p.seriesName}&nbsp;&nbsp;<b>${ampStr}</b>`;
                }
                return html;
            },
        },
        grid: {
            containLabel: false,
            left: '10%',
            right: '5%',
            top: '12%',
            bottom: '12%',
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
        // Zoom with scroll wheel
        dataZoom: [
            {
                type: 'inside',
                xAxisIndex: 0,
                filterMode: 'none',
            },
        ],
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

    const series = f.spectrum.map((magnitudes: number[], i: number) => ({
        name: f.labels?.[i] || `comp${i}`,
        data: magnitudes.map((mag: number, fi: number) => [f.freqAxis[fi], mag]),
        lineStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length], width: 2 },
        itemStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length] },
        // Item 6: Peak markers
        markPoint: {
            data: (f.peaks || [])
                .filter((p: { component: number }) => p.component === i)
                .map((p: { freqGHz: number; amplitude: number }) => ({
                    coord: [p.freqGHz, p.amplitude],
                    symbol: 'pin',
                    symbolSize: 30,
                    label: {
                        show: true,
                        formatter: `${p.freqGHz.toFixed(2)}`,
                        fontSize: 9,
                        color: '#fff',
                    },
                    itemStyle: { color: SERIES_COLORS[i % SERIES_COLORS.length] },
                })),
            animation: false,
        },
    }));

    spectrumChart.setOption({ series });
}

// --- Spectrogram (items 1, 2) ---

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
            formatter: (params: { value: number[] }) => {
                const [ti, fi, val] = params.value;
                return `${val.toFixed(1)} dB`;
            },
        },
        grid: {
            containLabel: false,
            left: '10%',
            right: '12%',
            top: '5%',
            bottom: '15%',
        },
        // Item 1: X axis = Time, Y axis = Frequency
        xAxis: {
            name: 'Time (ns)',
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
            name: 'Frequency (GHz)',
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
        // Item 2: dB scale
        visualMap: {
            min: -60,
            max: 0,
            calculable: true,
            orient: 'vertical',
            right: 0,
            top: '5%',
            bottom: '15%',
            text: ['0 dB', '-60 dB'],
            textStyle: { color: THEME.text2, fontSize: 10 },
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

    // Item 1: Axes swapped — X=time, Y=freq
    const timeLabels = (f.spectrogramTimes || []).map((t: number) => (t * 1e9).toFixed(2));
    const freqLabels = f.freqAxis.map((freq: number) => freq.toFixed(1));

    // Item 2: Convert to dB scale (relative to max)
    let maxVal = 0;
    for (let ti = 0; ti < f.spectrogram.length; ti++) {
        for (let fi = 0; fi < f.spectrogram[ti].length; fi++) {
            if (f.spectrogram[ti][fi] > maxVal) maxVal = f.spectrogram[ti][fi];
        }
    }

    const heatData: number[][] = [];
    const ref = maxVal > 0 ? maxVal : 1;
    for (let ti = 0; ti < f.spectrogram.length; ti++) {
        for (let fi = 0; fi < f.spectrogram[ti].length; fi++) {
            const val = f.spectrogram[ti][fi];
            // 20·log10(val/ref), clamped to [-60, 0]
            const dB = val > 0 ? Math.max(-60, 20 * Math.log10(val / ref)) : -60;
            // Heatmap data: [timeIndex, freqIndex, dB_value]
            heatData.push([ti, fi, dB]);
        }
    }

    spectrogramChart.setOption({
        xAxis: { data: timeLabels },
        yAxis: { data: freqLabels },
        visualMap: { min: -60, max: 0 },
        series: [{ data: heatData }],
    });
}
