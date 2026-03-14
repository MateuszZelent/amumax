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
		left: 8px;
		top: 8px;
		z-index: 10;
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
		border-radius: 6px;
		background: rgba(30, 32, 48, 0.9);
		border: 1px solid #3b3d4a;
		color: #888;
		font-size: 16px;
		cursor: pointer;
		transition: all 0.15s;
		backdrop-filter: blur(8px);
	}
	.toggle-btn:hover {
		background: #2a2d40;
		color: #fff;
	}
	.toolbar-content {
		margin-top: 4px;
		background: rgba(24, 26, 38, 0.95);
		border: 1px solid #3b3d4a;
		border-radius: 8px;
		padding: 10px;
		min-width: 160px;
		display: flex;
		flex-direction: column;
		gap: 10px;
		backdrop-filter: blur(8px);
	}
	.control-group {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.control-label {
		font-size: 11px;
		color: #999;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}
	.control-value {
		color: #ccc;
		font-weight: 500;
	}
	.slider {
		width: 100%;
		height: 4px;
		-webkit-appearance: none;
		appearance: none;
		background: #3b3d4a;
		border-radius: 2px;
		outline: none;
		cursor: pointer;
	}
	.slider::-webkit-slider-thumb {
		-webkit-appearance: none;
		width: 14px;
		height: 14px;
		border-radius: 50%;
		background: #6366f1;
		cursor: pointer;
	}
	.btn-group {
		display: flex;
		border-radius: 6px;
		overflow: hidden;
		border: 1px solid #3b3d4a;
	}
	.seg-btn {
		flex: 1;
		padding: 4px 0;
		font-size: 11px;
		font-weight: 500;
		color: #888;
		background: #1e2030;
		border: none;
		cursor: pointer;
		transition: all 0.15s;
	}
	.seg-btn:not(:last-child) {
		border-right: 1px solid #3b3d4a;
	}
	.seg-btn:hover {
		background: #2a2d40;
		color: #ccc;
	}
	.seg-btn.active {
		background: #4f46e5;
		color: #fff;
	}
	.action-btn {
		padding: 5px 0;
		font-size: 11px;
		font-weight: 500;
		color: #aaa;
		background: #1e2030;
		border: 1px solid #3b3d4a;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s;
	}
	.action-btn:hover {
		background: #2a2d40;
		color: #fff;
	}
</style>
