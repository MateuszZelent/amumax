<script lang="ts">
	import { previewState as p } from '$api/incoming/preview';
	import {
		brightness,
		qualityLevel,
		renderMode,
		resetCamera,
		setBrightness,
		setQuality,
		setRenderMode,
		setVoxelColorMode,
		setVoxelGap,
		setVoxelOpacity,
		setVoxelSampling,
		setVoxelThreshold,
		voxelColorMode,
		voxelGap,
		voxelOpacity,
		voxelSampling,
		voxelThreshold,
		type Preview3DRenderMode,
		type QualityLevel,
		type VoxelColorMode,
		type VoxelSampling
	} from '$lib/preview/preview3D';

	let expanded = false;
	let brightnessVal: number;
	let opacityVal: number;
	let gapVal: number;
	let thresholdVal: number;

	$: brightnessVal = $brightness;
	$: opacityVal = $voxelOpacity;
	$: gapVal = $voxelGap;
	$: thresholdVal = $voxelThreshold;

	const qualityLevels: { key: QualityLevel; label: string }[] = [
		{ key: 'low', label: 'LOW' },
		{ key: 'high', label: 'HIGH' },
		{ key: 'ultra', label: 'ULTRA' }
	];

	const renderModes: { key: Preview3DRenderMode; label: string }[] = [
		{ key: 'glyph', label: 'ARROWS' },
		{ key: 'voxel', label: 'VOXEL' }
	];

	const colorModes: { key: VoxelColorMode; label: string }[] = [
		{ key: 'orientation', label: 'ORI' },
		{ key: 'x', label: 'X' },
		{ key: 'y', label: 'Y' },
		{ key: 'z', label: 'Z' }
	];

	const samplingModes: { key: VoxelSampling; label: string }[] = [
		{ key: 1, label: '1X' },
		{ key: 2, label: '2X' },
		{ key: 4, label: '4X' }
	];

	$: isVisible = $p.nComp === 3 && $p.type === '3D';
</script>

{#if isVisible}
	<div class="toolbar" class:expanded>
		<button class="toggle-btn" onclick={() => (expanded = !expanded)} title="3D Controls">⚙</button>

		{#if expanded}
			<div class="toolbar-content">
				<div class="control-group">
					<div class="control-label">Render mode</div>
					<div class="btn-group btn-group--wide">
						{#each renderModes as { key, label }}
							<button
								class="seg-btn"
								class:active={$renderMode === key}
								onclick={() => setRenderMode(key)}
							>
								{label}
							</button>
						{/each}
					</div>
				</div>

				<div class="control-group">
					<div class="control-label">
						Brightness
						<span class="control-value">{brightnessVal.toFixed(1)}</span>
					</div>
					<input
						aria-label="3D preview brightness"
						type="range"
						min="0.3"
						max="3.0"
						step="0.1"
						value={brightnessVal}
						oninput={(event) => setBrightness(parseFloat(event.currentTarget.value))}
						class="slider"
					/>
				</div>

				<div class="control-group">
					<div class="control-label">Quality</div>
					<div class="btn-group btn-group--wide">
						{#each qualityLevels as { key, label }}
							<button
								class="seg-btn"
								class:active={$qualityLevel === key}
								onclick={() => setQuality(key)}
							>
								{label}
							</button>
						{/each}
					</div>
				</div>

				{#if $renderMode === 'voxel'}
					<div class="control-group">
						<div class="control-label">Color by</div>
						<div class="btn-group">
							{#each colorModes as { key, label }}
								<button
									class="seg-btn"
									class:active={$voxelColorMode === key}
									onclick={() => setVoxelColorMode(key)}
								>
									{label}
								</button>
							{/each}
						</div>
					</div>

					<div class="control-group">
						<div class="control-label">
							Opacity
							<span class="control-value">{opacityVal.toFixed(2)}</span>
						</div>
						<input
							aria-label="Voxel opacity"
							type="range"
							min="0.15"
							max="0.95"
							step="0.01"
							value={opacityVal}
							oninput={(event) => setVoxelOpacity(parseFloat(event.currentTarget.value))}
							class="slider"
						/>
					</div>

					<div class="control-group">
						<div class="control-label">
							Spacing
							<span class="control-value">{Math.round(gapVal * 100)}%</span>
						</div>
						<input
							aria-label="Voxel spacing"
							type="range"
							min="0.02"
							max="0.42"
							step="0.01"
							value={gapVal}
							oninput={(event) => setVoxelGap(parseFloat(event.currentTarget.value))}
							class="slider"
						/>
					</div>

					<div class="control-group">
						<div class="control-label">
							Min strength
							<span class="control-value">{thresholdVal.toFixed(2)}</span>
						</div>
						<input
							aria-label="Voxel threshold"
							type="range"
							min="0"
							max="0.95"
							step="0.01"
							value={thresholdVal}
							oninput={(event) => setVoxelThreshold(parseFloat(event.currentTarget.value))}
							class="slider"
						/>
					</div>

					<div class="control-group">
						<div class="control-label">Sampling</div>
						<div class="btn-group btn-group--wide">
							{#each samplingModes as { key, label }}
								<button
									class="seg-btn"
									class:active={$voxelSampling === key}
									onclick={() => setVoxelSampling(key)}
								>
									{label}
								</button>
							{/each}
						</div>
					</div>
				{/if}

				<button class="action-btn" onclick={resetCamera}>Reset Camera</button>
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
		background: linear-gradient(180deg, rgba(12, 18, 31, 0.92), rgba(8, 12, 22, 0.92));
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-md);
		min-width: 220px;
		max-width: 240px;
		display: flex;
		flex-direction: column;
		gap: var(--space-md);
		backdrop-filter: blur(14px);
		box-shadow: 0 20px 50px rgba(0, 0, 0, 0.28);
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
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.control-value {
		color: var(--text-1);
		font-weight: 600;
		font-family: var(--font-mono);
	}

	.slider {
		width: 100%;
		height: 4px;
		-webkit-appearance: none;
		appearance: none;
		background: linear-gradient(90deg, rgba(87, 200, 182, 0.2), rgba(107, 167, 255, 0.24));
		border-radius: 999px;
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
		border: 2px solid rgba(9, 14, 24, 0.9);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.28);
		cursor: pointer;
	}

	.btn-group {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		border-radius: var(--radius-md);
		overflow: hidden;
		border: 1px solid var(--border);
		background: rgba(255, 255, 255, 0.03);
	}

	.btn-group--wide {
		grid-template-columns: repeat(3, minmax(0, 1fr));
	}

	.seg-btn {
		padding: 0.42rem 0;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.04em;
		color: var(--text-3);
		background: transparent;
		border: none;
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
	}

	.seg-btn:not(:last-child) {
		border-right: 1px solid var(--border);
	}

	.seg-btn:hover {
		background: rgba(107, 167, 255, 0.08);
		color: var(--text-2);
	}

	.seg-btn.active {
		background: linear-gradient(135deg, rgba(87, 200, 182, 0.92), rgba(56, 178, 162, 0.92));
		color: #08101d;
	}

	.action-btn {
		padding: 0.48rem 0;
		font-size: 11px;
		font-weight: 700;
		letter-spacing: 0.04em;
		color: var(--text-2);
		background: rgba(255, 255, 255, 0.035);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
	}

	.action-btn:hover {
		background: rgba(107, 167, 255, 0.08);
		border-color: var(--border-interactive);
		color: var(--text-1);
	}
	
	@media (max-width: 767px) {
		.toolbar-content {
			min-width: 196px;
			max-width: 212px;
		}
	}
</style>
