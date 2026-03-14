<script lang="ts">
	import { metricsState as m } from '$api/incoming/metrics';
	import { postResetError } from '$api/outgoing/metrics';
</script>

<section>
	<h2>Metrics</h2>

	{#if $m.error}
		<div class="error-state">
			<p class="error-msg">Error collecting metrics: {$m.error}</p>
			<button class="retry-btn" on:click={postResetError}>Retry</button>
		</div>
	{:else}
		<div class="metrics-grid">
			<!-- System -->
			<div class="metric-group">
				<div class="group-label">System</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">CPU</span>
						<span class="tile-value">{$m.cpuPercentTotal.toFixed(1)}%</span>
					</div>
					<div class="bar"><div class="bar-fill blue" style="width: {$m.cpuPercentTotal}%"></div></div>
				</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">RAM</span>
						<span class="tile-value">{$m.ramPercentTotal.toFixed(1)}%</span>
					</div>
					<div class="bar"><div class="bar-fill green" style="width: {$m.ramPercentTotal}%"></div></div>
				</div>
			</div>

			<!-- Simulation -->
			<div class="metric-group">
				<div class="group-label">Simulation (PID {$m.pid})</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">CPU</span>
						<span class="tile-value">{$m.cpuPercent.toFixed(1)}%</span>
					</div>
					<div class="bar"><div class="bar-fill blue" style="width: {Math.min($m.cpuPercent, 100)}%"></div></div>
				</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">RAM</span>
						<span class="tile-value">{$m.ramPercent.toFixed(1)}%</span>
					</div>
					<div class="bar"><div class="bar-fill green" style="width: {$m.ramPercent}%"></div></div>
				</div>
			</div>

			<!-- GPU -->
			<div class="metric-group">
				<div class="group-label">GPU — {$m.gpuName}</div>
				<div class="stat-row">
					<span class="stat-label">Temp</span>
					<span class="stat-value">{$m.gpuTemperature}°C</span>
				</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">Utilization</span>
						<span class="tile-value">{$m.gpuUtilizationPercent}%</span>
					</div>
					<div class="bar"><div class="bar-fill purple" style="width: {$m.gpuUtilizationPercent}%"></div></div>
				</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">Power</span>
						<span class="tile-value">{$m.gpuPowerDraw.toFixed(0)} W</span>
					</div>
					<div class="bar"><div class="bar-fill purple" style="width: {($m.gpuPowerDraw / $m.gpuPowerLimit) * 100}%"></div></div>
				</div>
				<div class="metric-tile">
					<div class="tile-header">
						<span class="tile-name">VRAM</span>
						<span class="tile-value">{($m.gpuVramUsed / 1024).toFixed(1)} / {($m.gpuVramTotal / 1024).toFixed(1)} GiB</span>
					</div>
					<div class="bar"><div class="bar-fill purple" style="width: {($m.gpuVramUsed / $m.gpuVramTotal) * 100}%"></div></div>
				</div>
			</div>
		</div>
	{/if}
</section>

<style>
	.metrics-grid {
		display: flex;
		flex-direction: column;
		gap: var(--space-lg);
	}
	.metric-group {
		display: flex;
		flex-direction: column;
		gap: var(--space-sm);
	}
	.group-label {
		font-size: 10px;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.1em;
	}
	.metric-tile {
		display: flex;
		flex-direction: column;
		gap: 3px;
	}
	.tile-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.tile-name {
		font-size: 12px;
		color: var(--text-2);
	}
	.tile-value {
		font-size: 12px;
		font-family: var(--font-mono);
		color: var(--text-1);
		font-weight: 500;
	}
	.bar {
		height: 4px;
		background: var(--surface-3);
		border-radius: 2px;
		overflow: hidden;
	}
	.bar-fill {
		height: 100%;
		border-radius: 2px;
		transition: width var(--duration-normal) var(--easing-default);
	}
	.bar-fill.blue   { background: var(--accent); }
	.bar-fill.green  { background: var(--success); }
	.bar-fill.purple { background: #8b5cf6; }

	.stat-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 2px 0;
	}
	.stat-label {
		font-size: 12px;
		color: var(--text-2);
	}
	.stat-value {
		font-size: 12px;
		font-family: var(--font-mono);
		color: var(--text-1);
	}

	/* Error state */
	.error-state {
		display: flex;
		flex-direction: column;
		gap: var(--space-sm);
		align-items: flex-start;
	}
	.error-msg {
		color: var(--danger);
		font-size: 13px;
	}
	.retry-btn {
		padding: 6px 14px;
		background: transparent;
		border: 1px solid var(--danger);
		border-radius: var(--radius-md);
		color: var(--danger);
		font-size: 13px;
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
	}
	.retry-btn:hover {
		background: rgba(239, 68, 68, 0.1);
	}
</style>
