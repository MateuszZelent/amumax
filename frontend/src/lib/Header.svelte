<script lang="ts">
	import { headerState } from '$api/incoming/header';
	import { connectionState } from '$api/websocket';
	import Button from '$lib/ui/Button.svelte';
	import ConnectionBadge from '$lib/ui/ConnectionBadge.svelte';
	import StatusBadge from '$lib/ui/StatusBadge.svelte';
	import { panelPreferences, setDensityMode } from '$lib/ui/preferences';

	const solverTone = $derived(
		$headerState.status === 'running'
			? 'success'
			: $headerState.status === 'paused'
				? 'warn'
				: 'danger'
	);

	const solverLabel = $derived.by(() => {
		if (!$headerState.status) {
			return 'Idle';
		}
		return $headerState.status[0].toUpperCase() + $headerState.status.slice(1);
	});

	function toggleDensity() {
		setDensityMode($panelPreferences.densityMode === 'compact' ? 'cozy' : 'compact');
	}
</script>

<header class="topbar">
	<div class="topbar__cluster topbar__cluster--status">
		<ConnectionBadge state={$connectionState} />
		<StatusBadge label={solverLabel} tone={solverTone} pulse={$headerState.status === 'running'} />
	</div>

	<div class="topbar__path">
		<div class="topbar__eyebrow">Workspace</div>
		<strong>{$headerState.path || 'No simulation file loaded'}</strong>
	</div>

	<div class="topbar__cluster topbar__cluster--meta">
		<div class="topbar__version">v{$headerState.version || 'dev'}</div>
		<Button
			variant="outline"
			size="sm"
			tone="info"
			onclick={toggleDensity}
			title="Toggle UI density"
		>
			Density: {$panelPreferences.densityMode}
		</Button>
	</div>
</header>

<style>
	.topbar {
		position: sticky;
		top: 0;
		z-index: var(--z-sticky);
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: 1rem;
		padding: 1rem 1.1rem;
		border: 1px solid var(--border-subtle);
		border-radius: calc(var(--radius-lg) + 0.15rem);
		background:
			linear-gradient(180deg, rgba(18, 29, 49, 0.96), rgba(10, 18, 30, 0.94)),
			var(--surface-glass);
		backdrop-filter: blur(18px);
		box-shadow: var(--shadow-panel);
	}

	.topbar__cluster {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.topbar__path {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	.topbar__eyebrow {
		font-size: 0.74rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.14em;
		color: var(--text-3);
	}

	.topbar__path strong {
		font-size: 1.04rem;
		font-weight: 700;
		letter-spacing: -0.02em;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.topbar__version {
		font-size: 0.85rem;
		color: var(--text-3);
	}

	@media (max-width: 1023px) {
		.topbar {
			grid-template-columns: 1fr;
		}

		.topbar__cluster--meta {
			justify-content: space-between;
		}
	}
</style>
