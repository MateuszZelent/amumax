<script lang="ts">
	import Slider from '$components/Slider.svelte';
	import { meshState } from '$api/incoming/mesh';
	import { previewState as p } from '$api/incoming/preview';
	import { postLayer, postAllLayers } from '$api/outgoing/preview';

	let isDisabled: boolean;

	$: isDisabled = $meshState.Nz < 2;
	$: values = Array.from({ length: $meshState.Nz }, (_, i) => i);
	$: layer = $p.layer;
	$: allLayers = $p.allLayers;

	function toggleAllLayers() {
		const newValue = !allLayers;
		postAllLayers(newValue);
	}
</script>

<div class="flex gap-1">
	<div class="flex-1">
		<Slider
			label="Z Layer"
			bind:value={layer}
			{values}
			onChangeFunction={postLayer}
			isDisabled={isDisabled || allLayers}
		/>
	</div>
	{#if !isDisabled}
		<button
			class="h-11 rounded-md border px-3 text-sm font-medium transition-colors
				{allLayers
				? 'border-blue-500 bg-blue-600 text-white'
				: 'border-gray-600 bg-gray-800 text-gray-400 hover:bg-gray-700 hover:text-gray-200'}"
			on:click={toggleAllLayers}
		>
			{allLayers ? 'ALL' : 'ONE'}
		</button>
	{/if}
</div>
