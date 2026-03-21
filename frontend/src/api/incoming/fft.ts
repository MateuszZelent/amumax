import { writable } from 'svelte/store';

export interface FftPeak {
	freqGHz: number;
	amplitude: number;
	component: number;
}

export interface FftData {
	enabled: boolean;
	freqAxis: number[];
	labels: string[];
	maxFreqGHz: number;
	sampleIntervalNs: number;
	spectrum: number[][];
	spectrogram: number[][];
	spectrogramTimes: number[];
	spectrogramComponent: number;
	segProgress: number;
	segDurationNs: number;
	segElapsedNs: number;
	totalSegments: number;
	peaks: FftPeak[];
}

export const fftState = writable<FftData>({
	enabled: false,
	freqAxis: [],
	labels: [],
	maxFreqGHz: 0,
	sampleIntervalNs: 0,
	spectrum: [],
	spectrogram: [],
	spectrogramTimes: [],
	spectrogramComponent: 0,
	segProgress: 0,
	segDurationNs: 0,
	segElapsedNs: 0,
	totalSegments: 0,
	peaks: []
});
