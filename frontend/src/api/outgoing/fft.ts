import { post } from '$api/post';

export function postFftComponent(component: number) {
	post('fft/component', { component });
}

export function postFftClear() {
	post('fft/clear', {});
}

export function postFftMaxFrequency(maxFreqGHz: number) {
	post('fft/max-frequency', { maxFreqGHz });
}
