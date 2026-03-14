<script lang="ts">
	import { tablePlotState } from '$api/incoming/table-plot';
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

	let autosaveDraft = $state('');
	let maxPointsDraft = $state('');
	let stepDraft = $state('');

	const columnOptions = $derived($tablePlotState.columns.map((column) => ({ value: column, label: column })));

	function submitAutoSave(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) {
			return;
		}
		postAutoSaveInterval(value);
		autosaveDraft = '';
	}

	function submitMaxPoints(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) {
			return;
		}
		postMaxPoints(value);
		maxPointsDraft = '';
	}

	function submitStep(event: Event) {
		const value = (event.currentTarget as HTMLInputElement).value.trim();
		if (!value) {
			return;
		}
		postStep(value);
		stepDraft = '';
	}
</script>

<Panel
	title="Table Plot"
	subtitle="Shared analytical chart surface with explicit axes and persistence controls."
	panelId="tableplot"
	eyebrow="Visualization"
>
	<svelte:fragment slot="actions">
		<StatusBadge label={`${$tablePlotState.data.length} points`} tone={$tablePlotState.data.length ? 'info' : 'default'} />
	</svelte:fragment>

	{#if $tablePlotState.data.length === 0}
		<EmptyState
			title="No table data yet"
			description="Use TableSave() or set a non-zero autosave interval to start filling the analytical plot."
			tone="warn"
		>
			<div class="table-plot__empty-action">
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
		<div class="table-plot__toolbar">
			<SelectField label="X axis" value={$tablePlotState.xColumn} options={columnOptions} onchange={postXColumn} />
			<SelectField label="Y axis" value={$tablePlotState.yColumn} options={columnOptions} onchange={postYColumn} />
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

		<div id="table-plot" class="table-plot__canvas"></div>
	{/if}
</Panel>

<style>
	.table-plot__toolbar {
		display: grid;
		grid-template-columns: repeat(5, minmax(0, 1fr));
		gap: 0.8rem;
	}

	.table-plot__canvas {
		width: 100%;
		height: var(--canvas-min-height);
		min-height: var(--canvas-min-height);
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(5, 9, 17, 0.65);
		overflow: hidden;
	}

	.table-plot__empty-action {
		width: min(24rem, 100%);
	}

	@media (max-width: 1279px) {
		.table-plot__toolbar {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 767px) {
		.table-plot__toolbar {
			grid-template-columns: 1fr;
		}
	}
</style>
