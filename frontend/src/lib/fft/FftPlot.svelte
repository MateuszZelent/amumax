<script lang="ts">
	import { onMount } from 'svelte';
	import { fftState } from '$api/incoming/fft';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import StatusBadge from '$lib/ui/StatusBadge.svelte';
	import { plotSpectrum, plotSpectrogram } from './fft-plot';

	let activeTab = $state<'spectrum' | 'spectrogram'>('spectrum');

	$effect(() => {
		if ($fftState.enabled && $fftState.spectrum && $fftState.spectrum.length > 0) {
			requestAnimationFrame(() => {
				plotSpectrum();
				plotSpectrogram();
			});
		}
	});

	onMount(() => {
		if ($fftState.enabled && $fftState.spectrum && $fftState.spectrum.length > 0) {
			plotSpectrum();
			plotSpectrogram();
		}
	});
</script>

<Panel
	title="FFT Analysis"
	subtitle="Real-time spectrum and spectrogram from tracked quantities."
	panelId="fft"
	eyebrow="Visualization"
>
	{#snippet actions()}
		<StatusBadge
			label={$fftState.enabled ? `${$fftState.freqAxis?.length ?? 0} bins` : 'Disabled'}
			tone={$fftState.enabled ? 'info' : 'default'}
		/>
	{/snippet}

	{#if !$fftState.enabled}
		<EmptyState
			title="FFT is disabled"
			description="Start amumax with --fft flag and use FftTrack(m, minGHz, maxGHz, dGHz) in your script."
			tone="warn"
		/>
	{:else if !$fftState.spectrum || $fftState.spectrum.length === 0}
		<EmptyState
			title="No FFT data yet"
			description="Use FftTrack(m, 0, 30, 0.1) to track a quantity, then run the simulation."
			tone="warn"
		/>
	{:else}
		<div class="fft__tabs">
			<button
				class="fft__tab"
				class:fft__tab--active={activeTab === 'spectrum'}
				onclick={() => (activeTab = 'spectrum')}
			>
				Spectrum
			</button>
			<button
				class="fft__tab"
				class:fft__tab--active={activeTab === 'spectrogram'}
				onclick={() => (activeTab = 'spectrogram')}
			>
				Spectrogram
			</button>
		</div>

		<div id="fft-spectrum" class="fft__canvas" style:display={activeTab === 'spectrum' ? 'block' : 'none'}></div>
		<div id="fft-spectrogram" class="fft__canvas" style:display={activeTab === 'spectrogram' ? 'block' : 'none'}></div>
	{/if}
</Panel>

<style>
	.fft__tabs {
		display: flex;
		gap: 0.25rem;
		padding: 0 0 0.6rem;
	}

	.fft__tab {
		padding: 0.35rem 0.9rem;
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--text-2);
		font-size: 0.78rem;
		font-weight: 600;
		letter-spacing: 0.04em;
		cursor: pointer;
		transition:
			background 0.15s,
			color 0.15s,
			border-color 0.15s;
	}

	.fft__tab:hover {
		background: rgba(255, 255, 255, 0.04);
		color: var(--text-1);
	}

	.fft__tab--active {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--surface-1);
	}

	.fft__canvas {
		width: 100%;
		height: var(--canvas-min-height);
		min-height: var(--canvas-min-height);
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(5, 9, 17, 0.65);
		overflow: hidden;
	}
</style>
