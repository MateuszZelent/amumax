<script lang="ts">
	import { onDestroy, onMount } from 'svelte';

	import { headerState } from '$api/incoming/header';
	import { connected, initializeWebSocket } from '$api/websocket';
	import { postBreak, postRun } from '$api/outgoing/solver';
	import Alert from '$lib/alerts/Alert.svelte';
	import Console from '$lib/Console.svelte';
	import Header from '$lib/Header.svelte';
	import Mesh from '$lib/Mesh.svelte';
	import Metrics from '$lib/Metrics.svelte';
	import Parameters from '$lib/Parameters.svelte';
	import Preview from '$lib/preview/Preview.svelte';
	import Solver from '$lib/Solver.svelte';
	import TablePlot from '$lib/table-plot/TablePlot.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { panelPreferences, setActiveCompactTab } from '$lib/ui/preferences';
	import type { CompactTab, WorkspaceMode } from '$lib/ui/types';

	import '../app.css';

	let workspaceMode = $state<WorkspaceMode>('wide');
	let showShortcuts = $state(false);

	const compactTabs: { id: CompactTab; label: string }[] = [
		{ id: 'visualization', label: 'Visuals' },
		{ id: 'controls', label: 'Controls' },
		{ id: 'inspectors', label: 'Inspectors' },
		{ id: 'diagnostics', label: 'Diagnostics' }
	];

	function syncWorkspaceMode() {
		if (window.innerWidth < 768) {
			workspaceMode = 'mobile';
			return;
		}

		if (window.innerWidth < 1024) {
			workspaceMode = 'compact';
			return;
		}

		workspaceMode = 'wide';
	}

	function requestPreviewMode(mode: 'inline' | 'popout' | 'fullscreen') {
		window.dispatchEvent(new CustomEvent('amumax:preview-mode', { detail: mode }));
	}

	function isTypingTarget(target: EventTarget | null) {
		const element = target as HTMLElement | null;
		if (!element) {
			return false;
		}
		return ['INPUT', 'TEXTAREA', 'SELECT'].includes(element.tagName) || element.isContentEditable;
	}

	function handleShortcuts(event: KeyboardEvent) {
		if (showShortcuts && event.key === 'Escape') {
			showShortcuts = false;
			return;
		}

		if (isTypingTarget(event.target)) {
			return;
		}

		if ((event.key === '?' || (event.key === '/' && event.shiftKey)) && !event.metaKey && !event.ctrlKey) {
			event.preventDefault();
			showShortcuts = !showShortcuts;
			return;
		}

		if (!$connected) {
			return;
		}

		if (event.code === 'Space') {
			event.preventDefault();
			if ($headerState.status === 'running') {
				postBreak();
			} else {
				postRun('1e-9');
			}
			return;
		}

		if (event.key === 'Escape') {
			postBreak();
			return;
		}

		if (event.key.toLowerCase() === 'f') {
			event.preventDefault();
			requestPreviewMode('fullscreen');
		}
	}

	onMount(() => {
		initializeWebSocket();
		syncWorkspaceMode();
		window.addEventListener('resize', syncWorkspaceMode);
		window.addEventListener('keydown', handleShortcuts);
	});

	onDestroy(() => {
		window.removeEventListener('resize', syncWorkspaceMode);
		window.removeEventListener('keydown', handleShortcuts);
	});
</script>

<Alert />

<div class="app-shell">
	<Header />

	{#if workspaceMode !== 'wide'}
		<div class="workspace-compact-tabs">
			{#each compactTabs as tab}
				<Button
					size="sm"
					variant={$panelPreferences.activeCompactTab === tab.id ? 'solid' : 'outline'}
					tone={$panelPreferences.activeCompactTab === tab.id ? 'accent' : 'info'}
					onclick={() => setActiveCompactTab(tab.id)}
				>
					{tab.label}
				</Button>
			{/each}
			<Button size="sm" variant="ghost" tone="info" onclick={() => (showShortcuts = true)}>Shortcuts</Button>
		</div>
	{/if}

	<div class="workspace">
		<!-- Row 1: Visualization -->
		<div class="zone-viz">
			<Preview />
		</div>
		<div class="zone-table">
			<TablePlot />
		</div>

		<!-- Row 2: Controls -->
		<div class="zone-solver">
			<Solver />
		</div>
		<div class="zone-mesh">
			<Mesh />
		</div>

		<!-- Row 3: Diagnostics -->
		<div class="zone-console">
			<Console />
		</div>
		<div class="zone-metrics">
			<Metrics />
		</div>

		<!-- Row 4: Inspector (full width) -->
		<div class="zone-params">
			<Parameters />
		</div>
	</div>

	{#if showShortcuts}
		<div class="shortcuts-overlay" role="dialog" aria-modal="true" aria-label="Keyboard shortcuts">
			<div class="shortcuts-card">
				<div class="shortcuts-card__header">
					<div>
						<p class="shortcuts-card__eyebrow">Keyboard</p>
						<h2>Shortcuts</h2>
					</div>
					<Button size="sm" variant="ghost" tone="info" onclick={() => (showShortcuts = false)}>Close</Button>
				</div>

				<div class="shortcuts-list">
					<div><kbd>Space</kbd><span>Run default runtime or break active run</span></div>
					<div><kbd>Esc</kbd><span>Break simulation or close this overlay</span></div>
					<div><kbd>F</kbd><span>Send Preview to fullscreen</span></div>
					<div><kbd>?</kbd><span>Toggle shortcuts overlay</span></div>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.shortcuts-overlay {
		position: fixed;
		inset: 0;
		z-index: var(--z-modal);
		display: grid;
		place-items: center;
		padding: 1rem;
		background: rgba(3, 7, 14, 0.72);
		backdrop-filter: blur(10px);
	}

	.shortcuts-card {
		width: min(32rem, 100%);
		display: grid;
		gap: 1rem;
		padding: 1.25rem;
		border-radius: calc(var(--radius-lg) + 0.1rem);
		border: 1px solid var(--border-subtle);
		background: linear-gradient(180deg, rgba(18, 28, 47, 0.98), rgba(8, 13, 23, 0.98));
		box-shadow: var(--shadow-panel);
	}

	.shortcuts-card__header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 1rem;
	}

	.shortcuts-card__eyebrow {
		margin: 0 0 0.25rem;
		font-size: 0.72rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.14em;
		color: var(--text-3);
	}

	.shortcuts-card h2 {
		margin: 0;
		font-size: 1.2rem;
	}

	.shortcuts-list {
		display: grid;
		gap: 0.8rem;
	}

	.shortcuts-list div {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.8rem;
		align-items: center;
	}

	kbd {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 3.1rem;
		padding: 0.4rem 0.65rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		background: rgba(255, 255, 255, 0.03);
		font-family: 'IBM Plex Mono', monospace;
		color: var(--text-1);
	}
</style>
