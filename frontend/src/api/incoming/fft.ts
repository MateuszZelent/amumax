import { writable } from "svelte/store";

export interface FftData {
    enabled: boolean;
    freqAxis: number[];
    labels: string[];
    spectrum: number[][];
    spectrogram: number[][];
    spectrogramTimes: number[];
    spectrogramComponent: number;
    segProgress: number;
    segDurationNs: number;
    segElapsedNs: number;
    totalSegments: number;
}

export const fftState = writable<FftData>({
    enabled: false,
    freqAxis: [],
    labels: [],
    spectrum: [],
    spectrogram: [],
    spectrogramTimes: [],
    spectrogramComponent: 0,
    segProgress: 0,
    segDurationNs: 0,
    segElapsedNs: 0,
    totalSegments: 0,
});
