<script lang="ts">
	import { metricsState as metrics } from '$api/incoming/metrics';
	import { postResetError } from '$api/outgoing/metrics';
	import Button from '$lib/ui/Button.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import MetricTile from '$lib/ui/MetricTile.svelte';
	import Panel from '$lib/ui/Panel.svelte';

	function pct(value: number) {
		return `${value.toFixed(1)}%`;
	}

	function gib(value: number) {
		return `${(value / 1024).toFixed(2)} GiB`;
	}
</script>

<Panel title="Metrics" subtitle="Host and simulation telemetry with compact signal-first cards." panelId="metrics" eyebrow="Diagnostics">
	{#if $metrics.error}
		<EmptyState
			title="Telemetry unavailable"
			description={`The backend could not collect system metrics: ${$metrics.error}`}
			tone="danger"
		>
			<Button variant="solid" tone="danger" onclick={postResetError}>Retry metrics</Button>
		</EmptyState>
	{:else}
		<div class="metrics-sections">
			<section class="metrics-section">
				<header>
					<h3>Host</h3>
					<p>Machine-wide resource pressure.</p>
				</header>
				<div class="metrics-grid">
					<MetricTile label="CPU total" value={pct($metrics.cpuPercentTotal)} progress={$metrics.cpuPercentTotal} tone="info" />
					<MetricTile label="RAM total" value={pct($metrics.ramPercentTotal)} progress={$metrics.ramPercentTotal} tone="accent" />
					<MetricTile label="GPU util." value={pct($metrics.gpuUtilizationPercent)} progress={$metrics.gpuUtilizationPercent} tone="info" />
					<MetricTile
						label="Power draw"
						value={`${$metrics.gpuPowerDraw.toFixed(1)} W`}
						detail={`${$metrics.gpuPowerLimit.toFixed(1)} W limit`}
						progress={$metrics.gpuPowerLimit > 0 ? ($metrics.gpuPowerDraw / $metrics.gpuPowerLimit) * 100 : 0}
						tone="warn"
					/>
				</div>
			</section>

			<section class="metrics-section">
				<header>
					<h3>Simulation</h3>
					<p>Process-scoped diagnostics and GPU residency.</p>
				</header>
				<div class="metrics-grid">
					<MetricTile label="PID" value={`${$metrics.pid}`} detail={$metrics.gpuName || 'No GPU name'} />
					<MetricTile label="CPU proc." value={pct($metrics.cpuPercent)} progress={$metrics.cpuPercent} tone="info" />
					<MetricTile label="RAM proc." value={pct($metrics.ramPercent)} progress={$metrics.ramPercent} tone="accent" />
					<MetricTile
						label="VRAM used"
						value={gib($metrics.gpuVramUsed)}
						detail={`${gib($metrics.gpuVramTotal)} total`}
						progress={$metrics.gpuVramTotal > 0 ? ($metrics.gpuVramUsed / $metrics.gpuVramTotal) * 100 : 0}
						tone={$metrics.gpuVramTotal > 0 && $metrics.gpuVramUsed / $metrics.gpuVramTotal > 0.8 ? 'warn' : 'default'}
					/>
					<MetricTile label="GPU temp." value={`${$metrics.gpuTemperature}°C`} detail={$metrics.gpuUUID || 'UUID unavailable'} tone={$metrics.gpuTemperature > 80 ? 'warn' : 'default'} />
				</div>
			</section>
		</div>
	{/if}
</Panel>

<style>
	.metrics-sections {
		display: grid;
		gap: 1rem;
	}

	.metrics-section {
		display: grid;
		gap: 0.8rem;
		padding: 0.2rem 0;
	}

	.metrics-section header {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.metrics-section h3 {
		margin: 0;
		font-size: 0.95rem;
	}

	.metrics-section p {
		margin: 0;
		color: var(--text-2);
		font-size: 0.88rem;
	}

	.metrics-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.8rem;
	}

	@media (max-width: 639px) {
		.metrics-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
