<script lang="ts">
	import { onMount } from 'svelte';
	import { tablePlotState } from '$api/incoming/table-plot';
	import { fftState } from '$api/incoming/fft';
	import {
		postAutoSaveInterval,
		postMaxPoints,
		postStep,
		postXColumn,
		postYColumn
	} from '$api/outgoing/table-plot';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import SelectField from '$lib/ui/SelectField.svelte';
	import StatusBadge from '$lib/ui/StatusBadge.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { plotTable } from '$lib/table-plot/table-plot';
	import { plotSpectrum, plotSpectrogram } from '$lib/fft/fft-plot';

	type PlotTab = 'table' | 'spectrum' | 'spectrogram';
	let activeTab = $state<PlotTab>('table');

	let autosaveDraft = $state('');
	let maxPointsDraft = $state('');
	let stepDraft = $state('');

	const columnOptions = $derived(
		$tablePlotState.columns.map((column) => ({ value: column, label: column }))
	);

	function submitAutoSave(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) return;
		postAutoSaveInterval(value);
		autosaveDraft = '';
	}

	function submitMaxPoints(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) return;
		postMaxPoints(value);
		maxPointsDraft = '';
	}

	function submitStep(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) return;
		postStep(value);
		stepDraft = '';
	}

	$effect(() => {
		if ($tablePlotState.data.length > 0 && activeTab === 'table') {
			requestAnimationFrame(() => plotTable());
		}
	});

	$effect(() => {
		if ($fftState.enabled && $fftState.spectrum && $fftState.spectrum.length > 0) {
			requestAnimationFrame(() => {
				plotSpectrum();
				plotSpectrogram();
			});
		}
	});

	onMount(() => {
		void plotTable();
		if ($fftState.enabled) {
			plotSpectrum();
			plotSpectrogram();
		}
	});
</script>

<Panel
	title="Plots"
	subtitle="Time-domain table data, FFT spectrum, and spectrogram."
	panelId="plots"
	eyebrow="Visualization"
>
	{#snippet actions()}
		{#if activeTab === 'table'}
			<StatusBadge
				label={`${$tablePlotState.data.length} points`}
				tone={$tablePlotState.data.length ? 'info' : 'default'}
			/>
		{:else}
			<StatusBadge
				label={$fftState.enabled ? `${$fftState.freqAxis?.length ?? 0} bins` : 'FFT off'}
				tone={$fftState.enabled ? 'info' : 'default'}
			/>
		{/if}
	{/snippet}

	<div class="plots__tabs">
		<button
			class="plots__tab"
			class:plots__tab--active={activeTab === 'table'}
			onclick={() => (activeTab = 'table')}
		>
			Table Plot
		</button>
		<button
			class="plots__tab"
			class:plots__tab--active={activeTab === 'spectrum'}
			onclick={() => (activeTab = 'spectrum')}
		>
			Spectrum
		</button>
		<button
			class="plots__tab"
			class:plots__tab--active={activeTab === 'spectrogram'}
			onclick={() => (activeTab = 'spectrogram')}
		>
			Spectrogram
		</button>
	</div>

	<!-- Table Plot tab -->
	<div style:display={activeTab === 'table' ? 'block' : 'none'}>
		{#if $tablePlotState.data.length === 0}
			<EmptyState
				title="No table data yet"
				description="Use TableSave() or set a non-zero autosave interval."
				tone="warn"
			>
				<div class="plots__empty-action">
					<TextField
						label="Autosave interval"
						value={autosaveDraft}
						placeholder={`${$tablePlotState.autoSaveInterval}`}
						unit="s"
						mono={true}
						oninput={(event) =>
							(autosaveDraft = (event.currentTarget as HTMLInputElement).value)}
						onchange={submitAutoSave}
					/>
				</div>
			</EmptyState>
		{:else}
			<div class="plots__toolbar">
				<SelectField
					label="X axis"
					value={$tablePlotState.xColumn}
					options={columnOptions}
					onchange={postXColumn}
				/>
				<SelectField
					label="Y axis"
					value={$tablePlotState.yColumn}
					options={columnOptions}
					onchange={postYColumn}
				/>
				<TextField
					label="Max points"
					value={maxPointsDraft}
					placeholder={`${$tablePlotState.maxPoints}`}
					mono={true}
					oninput={(event) =>
						(maxPointsDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitMaxPoints}
				/>
				<TextField
					label="Step"
					value={stepDraft}
					placeholder={`${$tablePlotState.step}`}
					mono={true}
					oninput={(event) =>
						(stepDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitStep}
				/>
				<TextField
					label="Autosave"
					value={autosaveDraft}
					placeholder={`${$tablePlotState.autoSaveInterval}`}
					unit="s"
					mono={true}
					oninput={(event) =>
						(autosaveDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitAutoSave}
				/>
			</div>
			<div id="table-plot" class="plots__canvas"></div>
		{/if}
	</div>

	<!-- Spectrum tab -->
	<div style:display={activeTab === 'spectrum' ? 'block' : 'none'}>
		{#if !$fftState.enabled}
			<EmptyState
				title="FFT is disabled"
				description="Start with --fft flag and use FftTrack(m, minGHz, maxGHz, dGHz)."
				tone="warn"
			/>
		{:else if !$fftState.spectrum || $fftState.spectrum.length === 0}
			<EmptyState
				title="No FFT data yet"
				description="Run the simulation to accumulate FFT data."
				tone="warn"
			/>
		{:else}
			<div id="fft-spectrum" class="plots__canvas"></div>
		{/if}
	</div>

	<!-- Spectrogram tab -->
	<div style:display={activeTab === 'spectrogram' ? 'block' : 'none'}>
		{#if !$fftState.enabled}
			<EmptyState
				title="FFT is disabled"
				description="Start with --fft flag and use FftTrack(m, minGHz, maxGHz, dGHz)."
				tone="warn"
			/>
		{:else if !$fftState.spectrogram || $fftState.spectrogram.length === 0}
			<div class="plots__progress-card">
				<div class="plots__progress-header">
					<span class="plots__progress-title">Accumulating segment…</span>
					<span class="plots__progress-pct">{Math.round($fftState.segProgress * 100)}%</span>
				</div>
				<div class="plots__progress-bar">
					<div class="plots__progress-fill" style:width={`${Math.round($fftState.segProgress * 100)}%`}></div>
				</div>
				<div class="plots__progress-stats">
					<span>{$fftState.segElapsedNs.toFixed(1)} / {$fftState.segDurationNs.toFixed(1)} ns</span>
					<span>{$fftState.totalSegments} segments done</span>
				</div>
			</div>
		{:else}
			<div id="fft-spectrogram" class="plots__canvas"></div>
		{/if}
	</div>
</Panel>

<style>
	.plots__tabs {
		display: flex;
		gap: 0.25rem;
		padding: 0 0 0.6rem;
	}

	.plots__tab {
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

	.plots__tab:hover {
		background: rgba(255, 255, 255, 0.04);
		color: var(--text-1);
	}

	.plots__tab--active {
		background: var(--accent);
		border-color: var(--accent);
		color: var(--surface-1);
	}

	.plots__toolbar {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 0.8rem;
	}

	.plots__canvas {
		width: 100%;
		height: var(--canvas-min-height);
		min-height: var(--canvas-min-height);
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(5, 9, 17, 0.65);
		overflow: hidden;
	}

	.plots__empty-action {
		width: min(24rem, 100%);
	}

	.plots__progress-card {
		padding: 1.5rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(5, 9, 17, 0.65);
	}

	.plots__progress-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.75rem;
	}

	.plots__progress-title {
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--text-2);
	}

	.plots__progress-pct {
		font-family: var(--font-mono);
		font-size: 0.95rem;
		font-weight: 700;
		color: var(--accent);
	}

	.plots__progress-bar {
		height: 6px;
		border-radius: 3px;
		background: var(--surface-3);
		overflow: hidden;
		margin-bottom: 0.75rem;
	}

	.plots__progress-fill {
		height: 100%;
		border-radius: 3px;
		background: linear-gradient(90deg, var(--accent), var(--info));
		transition: width 0.3s ease;
	}

	.plots__progress-stats {
		display: flex;
		justify-content: space-between;
		font-family: var(--font-mono);
		font-size: 0.75rem;
		color: var(--text-3);
	}

	@media (max-width: 1279px) {
		.plots__toolbar {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 767px) {
		.plots__toolbar {
			grid-template-columns: 1fr;
		}
	}
</style>
