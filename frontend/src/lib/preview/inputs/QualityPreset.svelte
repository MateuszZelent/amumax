<script lang="ts">
	import { qualityLevel, setQuality, type QualityLevel } from '$lib/preview/preview3D';

	const levels: { key: QualityLevel; label: string }[] = [
		{ key: 'low', label: 'LOW' },
		{ key: 'high', label: 'HIGH' },
		{ key: 'ultra', label: 'ULTRA' }
	];
</script>

<div
	class="flex h-11 w-full overflow-hidden rounded-md border border-gray-600 bg-gray-800"
>
	<span class="flex items-center px-3 text-sm text-gray-400">Quality</span>
	{#each levels as { key, label }}
		<button
			class="flex-1 text-sm font-medium transition-colors
				{$qualityLevel === key
				? 'bg-blue-600 text-white'
				: 'text-gray-400 hover:bg-gray-700 hover:text-gray-200'}"
			on:click={() => setQuality(key)}
		>
			{label}
		</button>
	{/each}
</div>
