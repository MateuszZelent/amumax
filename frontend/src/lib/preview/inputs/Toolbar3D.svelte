<script lang="ts">
	import { brightness, setBrightness, qualityLevel, setQuality, type QualityLevel } from '$lib/preview/preview3D';
	import { resetCamera } from '$lib/preview/preview3D';
	import { previewState as p } from '$api/incoming/preview';

	let expanded = false;
	let brightnessVal: number;
	$: brightnessVal = $brightness;

	const levels: { key: QualityLevel; label: string }[] = [
		{ key: 'low', label: 'LOW' },
		{ key: 'high', label: 'HIGH' },
		{ key: 'ultra', label: 'ULTRA' }
	];

	$: isVisible = $p.nComp === 3;
</script>

{#if isVisible}
<div class="toolbar" class:expanded>
	<!-- Toggle button -->
	<button
		class="toggle-btn"
		on:click={() => expanded = !expanded}
		title="3D Controls"
	>
		⚙
	</button>

	{#if expanded}
		<div class="toolbar-content">
			<!-- Brightness -->
			<div class="control-group">
				<label class="control-label">
					Brightness
					<span class="control-value">{brightnessVal.toFixed(1)}</span>
				</label>
				<input
					type="range"
					min="0.3"
					max="3.0"
					step="0.1"
					value={brightnessVal}
					on:input={(e) => setBrightness(parseFloat(e.currentTarget.value))}
					class="slider"
				/>
			</div>

			<!-- Quality -->
			<div class="control-group">
				<label class="control-label">Quality</label>
				<div class="btn-group">
					{#each levels as { key, label }}
						<button
							class="seg-btn"
							class:active={$qualityLevel === key}
							on:click={() => setQuality(key)}
						>
							{label}
						</button>
					{/each}
				</div>
			</div>

			<!-- Reset Camera -->
			<button class="action-btn" on:click={resetCamera}>
				Reset Camera
			</button>
		</div>
	{/if}
</div>
{/if}

<style>
	.toolbar {
		position: absolute;
		left: var(--space-sm);
		top: var(--space-sm);
		z-index: var(--z-sticky);
		display: flex;
		flex-direction: column;
		gap: 0;
	}
	.toggle-btn {
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: var(--radius-md);
		background: var(--surface-glass);
		border: 1px solid var(--border);
		color: var(--text-3);
		font-size: 16px;
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
		backdrop-filter: blur(8px);
	}
	.toggle-btn:hover {
		background: var(--surface-3);
		color: var(--text-1);
	}
	.toolbar-content {
		margin-top: var(--space-xs);
		background: var(--surface-glass);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-md);
		min-width: 160px;
		display: flex;
		flex-direction: column;
		gap: var(--space-md);
		backdrop-filter: blur(12px);
	}
	.control-group {
		display: flex;
		flex-direction: column;
		gap: var(--space-xs);
	}
	.control-label {
		font-size: 11px;
		color: var(--text-2);
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.control-value {
		color: var(--text-1);
		font-weight: 500;
		font-family: var(--font-mono);
	}
	.slider {
		width: 100%;
		height: 4px;
		-webkit-appearance: none;
		appearance: none;
		background: var(--border);
		border-radius: 2px;
		outline: none;
		cursor: pointer;
		border: none;
	}
	.slider::-webkit-slider-thumb {
		-webkit-appearance: none;
		width: 14px;
		height: 14px;
		border-radius: 50%;
		background: var(--accent);
		cursor: pointer;
		transition: background var(--duration-fast) var(--easing-default);
	}
	.slider::-webkit-slider-thumb:hover {
		background: var(--accent-hover);
	}
	.btn-group {
		display: flex;
		border-radius: var(--radius-md);
		overflow: hidden;
		border: 1px solid var(--border);
	}
	.seg-btn {
		flex: 1;
		padding: 4px 0;
		font-size: 11px;
		font-weight: 500;
		color: var(--text-3);
		background: var(--surface-2);
		border: none;
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
	}
	.seg-btn:not(:last-child) {
		border-right: 1px solid var(--border);
	}
	.seg-btn:hover {
		background: var(--surface-3);
		color: var(--text-2);
	}
	.seg-btn.active {
		background: var(--accent);
		color: var(--text-1);
	}
	.action-btn {
		padding: 5px 0;
		font-size: 11px;
		font-weight: 500;
		color: var(--text-2);
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
	}
	.action-btn:hover {
		background: var(--surface-3);
		color: var(--text-1);
	}
</style>
