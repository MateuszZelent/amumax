<script lang="ts">
	import { tablePlotState } from '$api/incoming/table-plot';
	import { onMount, onDestroy } from 'svelte';
	import { resizeECharts } from './table-plot';
	import MaxPoints from './inputs/MaxPoints.svelte';
	import AutoSaveInterval from './inputs/AutoSaveInterval.svelte';
	import XColumn from './inputs/XColumn.svelte';
	import YColumn from './inputs/YColumn.svelte';
	import Step from './inputs/Step.svelte';

	let resizeObserver: ResizeObserver | null = null;

	onMount(() => {
		resizeECharts();

		// Use ResizeObserver on container instead of window resize
		const container = document.getElementById('table-plot');
		if (container) {
			resizeObserver = new ResizeObserver(() => {
				resizeECharts();
			});
			resizeObserver.observe(container);
		}
	});

	onDestroy(() => {
		if (resizeObserver) {
			resizeObserver.disconnect();
			resizeObserver = null;
		}
	});
</script>

<section>
	<h2 class="mb-4 text-2xl font-semibold">Table Plot</h2>
	{#if $tablePlotState.data.length === 0}
		<div class="msg">
			<p>
				No table data, use <code>TableSave()</code> or set a non-zero autosave interval to save data.
			</p>
			<div class="mt-5 w-72">
				<AutoSaveInterval />
			</div>
		</div>
	{:else}
		<div class="m-1 flex flex-wrap" id="parent-fields">
			<div class="basis-1/2">
				<XColumn />
			</div>
			<div class="basis-1/2">
				<YColumn />
			</div>
			<div class="basis-4/12">
				<MaxPoints />
			</div>
			<div class="basis-2/12">
				<Step />
			</div>
			<div class="basis-6/12">
				<AutoSaveInterval />
			</div>
		</div>

		<hr />
		<div id="table-plot"></div>
	{/if}
</section>

<style>
	section {
	}
	#table-plot {
		width: 100%;
		height: 500px;
	}
	.msg {
		color: var(--text-3);
	}
	#parent-fields > div {
		@apply p-1;
	}
</style>
