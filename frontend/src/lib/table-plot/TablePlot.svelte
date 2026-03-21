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
	import { postFftComponent, postFftMaxFrequency } from '$api/outgoing/fft';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import SelectField from '$lib/ui/SelectField.svelte';
	import StatusBadge from '$lib/ui/StatusBadge.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { plotTable } from '$lib/table-plot/table-plot';
	import { plotSpectrum, plotSpectrogram } from '$lib/fft/fft-plot';
	import {
		plotCorePosition,
		setOrbitWindow,
		getOrbitWindow,
		setEqualAspect,
		getEqualAspect,
		getMetrics,
		type CoreMetrics
	} from '$lib/core-position/core-position-plot';

	type PlotTab = 'table' | 'spectrum' | 'spectrogram' | 'corepos';
	let activeTab = $state<PlotTab>('table');
	let coreAspectEqual = $state(true);
	let coreOrbitNs = $state(2);
	let coreMetrics = $state<CoreMetrics>({
		radiusNm: 0, centerXNm: 0, centerYNm: 0,
		frequencyGHz: 0, totalPoints: 0, orbitPoints: 0, hasData: false,
	});

	let autosaveDraft = $state('');
	let maxPointsDraft = $state('');
	let stepDraft = $state('');
	let fftMaxFreqDraft = $state('');

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

	function submitFftMaxFreq(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) return;
		const parsed = Number.parseFloat(value);
		if (!Number.isFinite(parsed) || parsed <= 0) return;
		postFftMaxFrequency(parsed);
		fftMaxFreqDraft = '';
	}

	$effect(() => {
		if ($tablePlotState.data.length > 0 && activeTab === 'table') {
			requestAnimationFrame(() => plotTable());
		}
	});

	$effect(() => {
		if ($tablePlotState.data.length > 0 && activeTab === 'corepos') {
			requestAnimationFrame(() => {
				plotCorePosition();
				coreMetrics = getMetrics();
			});
		}
	});

	function handleOrbitWindow(ns: number) {
		coreOrbitNs = ns;
		setOrbitWindow(ns);
		coreMetrics = getMetrics();
	}

	function handleAspectToggle() {
		coreAspectEqual = !coreAspectEqual;
		setEqualAspect(coreAspectEqual);
	}

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
		<button
			class="plots__tab"
			class:plots__tab--active={activeTab === 'corepos'}
			onclick={() => (activeTab = 'corepos')}
		>
			Core Position
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
						oninput={(event) => (autosaveDraft = (event.currentTarget as HTMLInputElement).value)}
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
					oninput={(event) => (maxPointsDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitMaxPoints}
				/>
				<TextField
					label="Step"
					value={stepDraft}
					placeholder={`${$tablePlotState.step}`}
					mono={true}
					oninput={(event) => (stepDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitStep}
				/>
				<TextField
					label="Autosave"
					value={autosaveDraft}
					placeholder={`${$tablePlotState.autoSaveInterval}`}
					unit="s"
					mono={true}
					oninput={(event) => (autosaveDraft = (event.currentTarget as HTMLInputElement).value)}
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
		{:else}
			<div class="plots__toolbar plots__toolbar--narrow">
				<TextField
					label="f_max"
					value={fftMaxFreqDraft}
					placeholder={$fftState.maxFreqGHz > 0 ? `${$fftState.maxFreqGHz.toFixed(2)}` : ''}
					unit="GHz"
					hint={$fftState.sampleIntervalNs > 0
						? `Sampling every ${$fftState.sampleIntervalNs.toFixed(3)} ns`
						: 'Sampling interval unavailable'}
					mono={true}
					oninput={(event) => (fftMaxFreqDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitFftMaxFreq}
				/>
			</div>
			{#if !$fftState.spectrum || $fftState.spectrum.length === 0}
				<EmptyState
					title="No FFT data yet"
					description="Run the simulation to accumulate FFT data."
					tone="warn"
				/>
			{:else}
				<div id="fft-spectrum" class="plots__canvas"></div>
			{/if}
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
		{:else}
			<div class="plots__toolbar plots__toolbar--narrow">
				<TextField
					label="f_max"
					value={fftMaxFreqDraft}
					placeholder={$fftState.maxFreqGHz > 0 ? `${$fftState.maxFreqGHz.toFixed(2)}` : ''}
					unit="GHz"
					hint={$fftState.sampleIntervalNs > 0
						? `Sampling every ${$fftState.sampleIntervalNs.toFixed(3)} ns`
						: 'Sampling interval unavailable'}
					mono={true}
					oninput={(event) => (fftMaxFreqDraft = (event.currentTarget as HTMLInputElement).value)}
					onchange={submitFftMaxFreq}
				/>
				<SelectField
					label="Component"
					value={String($fftState.spectrogramComponent)}
					options={($fftState.labels || []).map((l, i) => ({ value: String(i), label: l }))}
					onchange={(v) => postFftComponent(Number.parseInt(v, 10))}
				/>
			</div>
			<!-- Segment progress – always visible while accumulating -->
		{@const pct = Math.round($fftState.segProgress * 100)}
		{@const hasData = $fftState.spectrogram && $fftState.spectrogram.length > 0}
		<div class="seg-progress" class:seg-progress--standalone={!hasData}>
			<div class="seg-progress__row">
				<span class="seg-progress__label">
					{#if hasData}
						Segment {$fftState.totalSegments + 1}
					{:else}
						Accumulating first segment…
					{/if}
				</span>
				<span class="seg-progress__right">
					<span class="seg-progress__time">{$fftState.segElapsedNs.toFixed(1)} / {$fftState.segDurationNs.toFixed(1)} ns</span>
					<span class="seg-progress__pct">{pct}%</span>
				</span>
			</div>
			<div class="seg-progress__track">
				<div class="seg-progress__fill" style:width={`${pct}%`}></div>
				<div class="seg-progress__glow" style:width={`${pct}%`}></div>
			</div>
			{#if hasData}
				<span class="seg-progress__count">{$fftState.totalSegments} segments completed</span>
			{/if}
		</div>

		{#if hasData}
			<div id="fft-spectrogram" class="plots__canvas"></div>
		{/if}
		{/if}
	</div>

	<!-- Core Position tab -->
	<div style:display={activeTab === 'corepos' ? 'block' : 'none'}>
		{#if !$tablePlotState.columns.includes('ext_coreposx') || !$tablePlotState.columns.includes('ext_coreposy')}
			<EmptyState
				title="Core position data unavailable"
				description="Add ext_coreposx and ext_coreposy to your table output using TableAdd(ext_coreposx) and TableAdd(ext_coreposy)."
				tone="warn"
			/>
		{:else if $tablePlotState.data.length === 0}
			<EmptyState
				title="No table data yet"
				description="Use TableSave() or set a non-zero autosave interval."
				tone="warn"
			/>
		{:else}
			<!-- Inline controls -->
			<div class="cp__controls">
				<div class="cp__group">
					<span class="cp__label">Orbit window</span>
					<div class="cp__seg">
						{#each [1, 2, 5, 10, 20] as ns}
							<button
								class="cp__seg-btn"
								class:cp__seg-btn--active={coreOrbitNs === ns}
								onclick={() => handleOrbitWindow(ns)}
							>{ns} ns</button>
						{/each}
					</div>
				</div>
				<button
					class="cp__seg-btn"
					class:cp__seg-btn--active={coreAspectEqual}
					onclick={handleAspectToggle}
					title="Equal aspect ratio"
				>1∶1</button>
			</div>

			<div id="core-position-plot" class="plots__canvas"></div>

			<!-- Metrics panel -->
			{#if coreMetrics.hasData}
				<div class="cp__metrics">
					<div class="cp__metric cp__metric--primary">
						<span class="cp__metric-value">{coreMetrics.radiusNm.toFixed(2)}</span>
						<span class="cp__metric-unit">nm</span>
						<span class="cp__metric-label">Orbit radius</span>
					</div>
					<div class="cp__metric">
						<span class="cp__metric-value">{coreMetrics.frequencyGHz.toFixed(3)}</span>
						<span class="cp__metric-unit">GHz</span>
						<span class="cp__metric-label">Frequency</span>
					</div>
					<div class="cp__metric">
						<span class="cp__metric-value">{coreMetrics.centerXNm.toFixed(2)}</span>
						<span class="cp__metric-unit">nm</span>
						<span class="cp__metric-label">Center X</span>
					</div>
					<div class="cp__metric">
						<span class="cp__metric-value">{coreMetrics.centerYNm.toFixed(2)}</span>
						<span class="cp__metric-unit">nm</span>
						<span class="cp__metric-label">Center Y</span>
					</div>
					<div class="cp__metric">
						<span class="cp__metric-value">{coreMetrics.orbitPoints}</span>
						<span class="cp__metric-unit">/ {coreMetrics.totalPoints}</span>
						<span class="cp__metric-label">Orbit / Total pts</span>
					</div>
				</div>
			{/if}
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

	.plots__toolbar--narrow {
		grid-template-columns: minmax(0, 10rem) auto;
		align-items: end;
		margin-bottom: 0.6rem;
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

	/* ── Segment progress ── */

	.seg-progress {
		padding: 0.5rem 0.75rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(5, 9, 17, 0.45);
		margin-bottom: 0.5rem;
	}

	.seg-progress--standalone {
		padding: 1.5rem;
		background: rgba(5, 9, 17, 0.65);
	}

	.seg-progress__row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.4rem;
	}

	.seg-progress__label {
		font-size: 0.78rem;
		font-weight: 600;
		color: var(--text-2);
	}

	.seg-progress__right {
		display: flex;
		align-items: center;
		gap: 0.6rem;
	}

	.seg-progress__time {
		font-family: var(--font-mono);
		font-size: 0.72rem;
		color: var(--text-3);
	}

	.seg-progress__pct {
		font-family: var(--font-mono);
		font-size: 0.85rem;
		font-weight: 700;
		color: var(--accent);
		min-width: 2.5rem;
		text-align: right;
	}

	.seg-progress__track {
		position: relative;
		height: 4px;
		border-radius: 2px;
		background: var(--surface-3);
		overflow: hidden;
	}

	.seg-progress--standalone .seg-progress__track {
		height: 6px;
		border-radius: 3px;
		margin-bottom: 0.6rem;
	}

	.seg-progress__fill {
		position: absolute;
		top: 0;
		left: 0;
		height: 100%;
		border-radius: inherit;
		background: linear-gradient(90deg, var(--accent), var(--info));
		transition: width 0.3s ease;
	}

	.seg-progress__glow {
		position: absolute;
		top: -2px;
		left: 0;
		height: calc(100% + 4px);
		border-radius: inherit;
		background: linear-gradient(90deg, var(--accent), var(--info));
		opacity: 0.35;
		filter: blur(4px);
		transition: width 0.3s ease;
		pointer-events: none;
	}

	.seg-progress__count {
		display: block;
		margin-top: 0.35rem;
		font-family: var(--font-mono);
		font-size: 0.7rem;
		color: var(--text-3);
		text-align: right;
	}

	/* ── Core Position ── */

	.cp__controls {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		margin-bottom: 0.5rem;
	}

	.cp__group {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.cp__label {
		font-size: 0.72rem;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.cp__seg {
		display: flex;
		gap: 2px;
		background: var(--surface-2);
		border-radius: var(--radius-pill);
		padding: 2px;
	}

	.cp__seg-btn {
		padding: 0.2rem 0.6rem;
		border: none;
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--text-3);
		font-size: 0.7rem;
		font-weight: 600;
		font-family: var(--font-mono);
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
	}

	.cp__seg-btn:hover {
		color: var(--text-1);
		background: rgba(255, 255, 255, 0.05);
	}

	.cp__seg-btn--active {
		background: var(--accent);
		color: var(--surface-1);
	}

	.cp__metrics {
		display: grid;
		grid-template-columns: repeat(5, 1fr);
		gap: 1px;
		margin-top: 0.5rem;
		border-radius: var(--radius-md);
		overflow: hidden;
		border: 1px solid var(--border-subtle);
		background: var(--border-subtle);
	}

	.cp__metric {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.1rem;
		padding: 0.6rem 0.4rem;
		background: var(--surface-1);
	}

	.cp__metric--primary {
		background: rgba(52, 211, 153, 0.08);
	}

	.cp__metric-value {
		font-family: var(--font-mono);
		font-size: 0.95rem;
		font-weight: 700;
		color: var(--text-1);
		line-height: 1;
	}

	.cp__metric--primary .cp__metric-value {
		color: #34d399;
	}

	.cp__metric-unit {
		font-family: var(--font-mono);
		font-size: 0.65rem;
		color: var(--text-3);
		line-height: 1;
	}

	.cp__metric-label {
		font-size: 0.62rem;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.05em;
		line-height: 1;
		margin-top: 0.15rem;
	}

	@media (max-width: 1279px) {
		.plots__toolbar {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.cp__metrics {
			grid-template-columns: repeat(3, 1fr);
		}
	}

	@media (max-width: 767px) {
		.plots__toolbar {
			grid-template-columns: 1fr;
		}

		.cp__metrics {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>
