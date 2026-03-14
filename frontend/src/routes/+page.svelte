<script lang="ts">
	import Header from '$lib/Header.svelte';
	import Display from '$lib/preview/Preview.svelte';
	import TablePlot from '$lib/table-plot/TablePlot.svelte';
	import Solver from '$lib/Solver.svelte';
	import Console from '$lib/Console.svelte';
	import Mesh from '$lib/Mesh.svelte';
	import Parameters from '$lib/Parameters.svelte';
	import Metrics from '$lib/Metrics.svelte';

	import '../app.css';
	import Alert from '$lib/alerts/Alert.svelte';
	import { onMount } from 'svelte';
	import { initializeWebSocket, connected } from '$api/websocket';
	import { postRun, postBreak, postRelax, postMinimize } from '$api/outgoing/solver';

	onMount(initializeWebSocket);

	function handleKeydown(e: KeyboardEvent) {
		// Skip shortcuts when typing in inputs
		const tag = (e.target as HTMLElement)?.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

		if (e.ctrlKey || e.metaKey) {
			switch (e.key) {
				case 'Enter':
					e.preventDefault();
					postRun('1e-9');
					break;
				case '.':
					e.preventDefault();
					postBreak();
					break;
				case 'r':
					e.preventDefault();
					postRelax();
					break;
				case 'm':
					e.preventDefault();
					postMinimize();
					break;
			}
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<Alert />
<Header />
<div class="workspace">
	<!-- Visualization zone -->
	<div class="zone-viz">
		<Display />
	</div>

	<!-- Controls zone -->
	<div class="zone-controls">
		<TablePlot />
		<Solver />
		<Mesh />
	</div>

	<!-- Diagnostics zone -->
	<div class="zone-diag">
		<Console />
		<Metrics />
	</div>

	<!-- Inspector zone -->
	<div class="zone-inspector">
		<Parameters />
	</div>
</div>

<!-- Disconnected overlay -->
{#if !$connected}
	<div class="disconnected-overlay">
		<div class="disconnected-card">
			<span class="pulse-dot"></span>
			<span class="disconnected-title">Disconnected</span>
			<span class="disconnected-sub">Attempting to reconnect…</span>
		</div>
	</div>
{/if}

<style>
	.workspace {
		display: grid;
		grid-template-columns: 1fr 1fr;
		grid-template-areas:
			'viz       controls'
			'diag      inspector';
		gap: var(--space-md);
		padding: var(--space-md);
		min-height: calc(100vh - 44px);
		box-sizing: border-box;
	}
	.zone-viz       { grid-area: viz; }
	.zone-controls  { grid-area: controls; }
	.zone-diag      { grid-area: diag; }
	.zone-inspector { grid-area: inspector; }

	/* Each zone is a flex column so children stack with consistent gap */
	.zone-viz,
	.zone-controls,
	.zone-diag,
	.zone-inspector {
		display: flex;
		flex-direction: column;
		gap: var(--space-md);
		min-width: 0;
	}

	/* ─── Tablet / narrow (<1024) ───────────────────────────────────── */
	@media (max-width: 1024px) {
		.workspace {
			grid-template-columns: 1fr;
			grid-template-areas:
				'viz'
				'controls'
				'diag'
				'inspector';
		}
	}

	/* ─── Section styling ───────────────────────────────────────────── */
	:global(section) {
		border: 1px solid var(--border);
		padding: var(--space-md);
		border-radius: var(--radius-lg);
		background-color: var(--surface-1);
		overflow: hidden;
		transition: border-color var(--duration-fast) var(--easing-default);
	}
	:global(section:hover) {
		border-color: var(--border-interactive);
	}
	:global(body) {
		font-family: var(--font-ui);
	}
	:global(section > h2) {
		color: var(--text-2);
		padding: var(--space-sm) 0;
		margin: 0 0 var(--space-md) 0;
		border-bottom: 1px solid var(--border);
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		background: none;
	}

	/* ─── Disconnected overlay ─────────────────────────────────────── */
	.disconnected-overlay {
		position: fixed;
		inset: 0;
		z-index: var(--z-modal);
		background: rgba(8, 13, 26, 0.75);
		display: flex;
		align-items: center;
		justify-content: center;
		backdrop-filter: blur(4px);
	}
	.disconnected-card {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-sm);
		padding: var(--space-xl) var(--space-2xl);
		background: var(--surface-1);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-lg);
	}
	.pulse-dot {
		width: 12px;
		height: 12px;
		border-radius: 50%;
		background: var(--danger);
		animation: pulse 1.5s ease-in-out infinite;
	}
	@keyframes pulse {
		0%, 100% { opacity: 1; transform: scale(1); }
		50% { opacity: 0.4; transform: scale(0.85); }
	}
	.disconnected-title {
		font-size: 16px;
		font-weight: 600;
		color: var(--text-1);
	}
	.disconnected-sub {
		font-size: 13px;
		color: var(--text-3);
	}
</style>
