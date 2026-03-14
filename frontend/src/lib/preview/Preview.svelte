<script lang="ts">
	import { connected } from '$api/websocket';
	import { meshState } from '$api/incoming/mesh';
	import { previewState } from '$api/incoming/preview';
	import {
		postAllLayers,
		postComponent,
		postLayer,
		postQuantity,
		postXChosenSize,
		postYChosenSize
	} from '$api/outgoing/preview';
	import Slider from '$components/Slider.svelte';
	import Button from '$lib/ui/Button.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import SelectField from '$lib/ui/SelectField.svelte';
	import SegmentedControl from '$lib/ui/SegmentedControl.svelte';
	import StatusBadge from '$lib/ui/StatusBadge.svelte';
	import { panelPreferences, setPreferredPreviewMode } from '$lib/ui/preferences';
	import type { SelectOption } from '$lib/ui/SelectField.svelte';
	import type { ViewportMode } from '$lib/ui/types';
	import { get } from 'svelte/store';
	import { onDestroy, onMount } from 'svelte';
	import { preview2D, resizeECharts } from './preview2D';
	import {
		preview3D,
		qualityLevel,
		renderMode,
		resetCamera,
		setQuality,
		setRenderMode,
		threeDPreview,
		type Preview3DRenderMode,
		type QualityLevel
	} from './preview3D';
	import Toolbar3D from './inputs/Toolbar3D.svelte';
	import { quantities } from './inputs/quantities';

	let viewMode = $state<ViewportMode>('inline');
	let previewWrapper: HTMLDivElement;

	let popX = $state(60);
	let popY = $state(60);
	let popW = $state(760);
	let popH = $state(540);
	let dragging = $state(false);
	let resizing = $state(false);
	let dragOffX = $state(0);
	let dragOffY = $state(0);

	const quantityOptions = $derived(
		Object.entries(quantities).flatMap(([group, items]) =>
			items.map((item) => ({
				value: item,
				label: item,
				group: group === 'Common' ? undefined : group
			} satisfies SelectOption))
		)
	);

	const componentOptions = $derived([
		{ value: '3D', label: '3D', disabled: $previewState.nComp === 1 },
		{ value: 'x', label: 'x', disabled: $previewState.nComp === 1 },
		{ value: 'y', label: 'y', disabled: $previewState.nComp === 1 },
		{ value: 'z', label: 'z', disabled: $previewState.nComp === 1 }
	]);

	const qualityOptions = $derived(
		(['low', 'high', 'ultra'] as QualityLevel[]).map((level) => ({
			value: level,
			label: level.toUpperCase()
		}))
	);

	const renderOptions = $derived(
		(['glyph', 'voxel'] as Preview3DRenderMode[]).map((mode) => ({
			value: mode,
			label: mode === 'glyph' ? 'Arrows' : 'Voxel'
		}))
	);

	const hasData = $derived(
		($previewState.scalarField?.length ?? 0) > 0 || ($previewState.vectorFieldPositions?.length ?? 0) > 0
	);
	const previewTone = $derived(!$connected ? 'warn' : hasData ? 'info' : 'default');

	$effect(() => {
		viewMode = $panelPreferences.preferredPreviewMode;
	});

	function setMode(mode: ViewportMode) {
		if (viewMode === 'fullscreen' && document.fullscreenElement) {
			document.exitFullscreen().catch(() => undefined);
		}

		if (mode === 'fullscreen' && previewWrapper) {
			previewWrapper.requestFullscreen().catch(() => undefined);
		}

		viewMode = mode;
		setPreferredPreviewMode(mode);
		scheduleResize();
	}

	function scheduleResize() {
		window.setTimeout(() => {
			const container = document.getElementById('container');
			const display = get(threeDPreview);
			if (display && container) {
				display.renderer.setSize(container.clientWidth, container.clientHeight);
				display.camera.aspect = container.clientWidth / container.clientHeight;
				display.camera.updateProjectionMatrix();
			}
			resizeECharts();
		}, 120);
	}

	function toggleAllLayers() {
		postAllLayers(!$previewState.allLayers);
	}

	function onFullscreenChange() {
		if (!document.fullscreenElement && viewMode === 'fullscreen') {
			viewMode = 'inline';
			setPreferredPreviewMode('inline');
		}
		scheduleResize();
	}

	function startDrag(event: MouseEvent) {
		if (resizing) {
			return;
		}
		dragging = true;
		dragOffX = event.clientX - popX;
		dragOffY = event.clientY - popY;
		event.preventDefault();
	}

	function startResize(event: MouseEvent) {
		resizing = true;
		dragOffX = event.clientX;
		dragOffY = event.clientY;
		event.preventDefault();
	}

	function onMouseMove(event: MouseEvent) {
		if (dragging) {
			popX = event.clientX - dragOffX;
			popY = event.clientY - dragOffY;
		} else if (resizing) {
			const dx = event.clientX - dragOffX;
			const dy = event.clientY - dragOffY;
			popW = Math.max(460, popW + dx);
			popH = Math.max(340, popH + dy);
			dragOffX = event.clientX;
			dragOffY = event.clientY;
		}
	}

	function onMouseUp() {
		if (dragging || resizing) {
			dragging = false;
			resizing = false;
			scheduleResize();
		}
	}

	function onPreviewModeRequest(event: Event) {
		const customEvent = event as CustomEvent<ViewportMode>;
		setMode(customEvent.detail);
	}

	async function renderCurrentPreview() {
		const state = get(previewState);
		if (state.type === '3D') {
			await preview3D();
			return;
		}

		await preview2D();
	}

	onMount(() => {
		document.addEventListener('fullscreenchange', onFullscreenChange);
		document.addEventListener('mousemove', onMouseMove);
		document.addEventListener('mouseup', onMouseUp);
		window.addEventListener('amumax:preview-mode', onPreviewModeRequest as EventListener);
		scheduleResize();
		void renderCurrentPreview();
	});

	onDestroy(() => {
		document.removeEventListener('fullscreenchange', onFullscreenChange);
		document.removeEventListener('mousemove', onMouseMove);
		document.removeEventListener('mouseup', onMouseUp);
		window.removeEventListener('amumax:preview-mode', onPreviewModeRequest as EventListener);
	});
</script>

<Panel
	title="Preview"
	subtitle="Primary simulation surface with consistent controls for 2D and 3D modes."
	panelId="preview"
	eyebrow="Visualization"
	tone={previewTone}
>
	{#snippet actions()}
		<StatusBadge label={$previewState.type || 'Awaiting data'} tone={hasData ? 'info' : 'default'} />
		<div class="preview-mode-switcher">
			<Button
				size="sm"
				variant={viewMode === 'inline' ? 'solid' : 'outline'}
				tone="info"
				onclick={() => setMode('inline')}
			>
				Dock
			</Button>
			<Button
				size="sm"
				variant={viewMode === 'popout' ? 'solid' : 'outline'}
				tone="info"
				onclick={() => setMode('popout')}
			>
				Popout
			</Button>
			<Button
				size="sm"
				variant={viewMode === 'fullscreen' ? 'solid' : 'outline'}
				tone="info"
				onclick={() => setMode(viewMode === 'fullscreen' ? 'inline' : 'fullscreen')}
			>
				{viewMode === 'fullscreen' ? 'Exit' : 'Fullscreen'}
			</Button>
		</div>
	{/snippet}

	<div class="preview-toolbar">
		<SelectField
			label="Quantity"
			value={$previewState.quantity}
			options={quantityOptions}
			onchange={postQuantity}
		/>
		<SegmentedControl
			label="Component"
			value={$previewState.component}
			options={componentOptions}
			onchange={postComponent}
		/>
		{#if $previewState.xPossibleSizes.length > 0}
			<Slider
				label="X data points"
				value={$previewState.xChosenSize}
				values={$previewState.xPossibleSizes}
				onChangeFunction={postXChosenSize}
			/>
		{/if}
		{#if $previewState.yPossibleSizes.length > 0}
			<Slider
				label="Y data points"
				value={$previewState.yChosenSize}
				values={$previewState.yPossibleSizes}
				onChangeFunction={postYChosenSize}
			/>
		{/if}
		<div class="preview-toolbar__stack">
			{#if $meshState.Nz > 1}
				<Slider
					label="Layer"
					value={$previewState.layer}
					values={Array.from({ length: $meshState.Nz }, (_, index) => index)}
					onChangeFunction={postLayer}
					isDisabled={$previewState.allLayers}
				/>
			{/if}
			<div class="preview-toolbar__actions">
				<Button
					variant="outline"
					tone="accent"
					onclick={resetCamera}
					disabled={$previewState.nComp !== 3 || $previewState.type !== '3D'}
				>
					Reset camera
				</Button>
				<Button
					variant={$previewState.allLayers ? 'solid' : 'outline'}
					tone="info"
					onclick={toggleAllLayers}
					disabled={$meshState.Nz < 2}
				>
					{$previewState.allLayers ? 'All layers' : 'Single layer'}
				</Button>
			</div>
		</div>
		<SegmentedControl
			label="Quality"
			value={$qualityLevel}
			options={qualityOptions}
			onchange={(next) => setQuality(next as QualityLevel)}
		/>
		{#if $previewState.type === '3D' && $previewState.nComp === 3}
			<SegmentedControl
				label="Render"
				value={$renderMode}
				options={renderOptions}
				onchange={(next) => setRenderMode(next as Preview3DRenderMode)}
			/>
		{/if}
	</div>

	<div
		class="preview-wrapper"
		class:preview-wrapper--popout={viewMode === 'popout'}
		class:preview-wrapper--fullscreen={viewMode === 'fullscreen'}
		style={viewMode === 'popout' ? `left:${popX}px;top:${popY}px;width:${popW}px;height:${popH}px;` : ''}
		bind:this={previewWrapper}
	>
		{#if viewMode === 'popout'}
			<div class="preview-wrapper__titlebar" role="presentation" onmousedown={startDrag}>
				<span>Floating preview</span>
				<div class="preview-wrapper__title-actions">
					<Button size="sm" variant="ghost" tone="info" onclick={() => setMode('inline')}>Dock</Button>
					<Button size="sm" variant="ghost" tone="info" onclick={() => setMode('fullscreen')}>Fullscreen</Button>
				</div>
			</div>
		{/if}

		<Toolbar3D />

		{#if !$connected || !hasData}
			<div class="preview-wrapper__empty">
				<EmptyState
					title={!$connected ? 'Preview offline' : 'No preview data yet'}
					description={!$connected
						? 'Reconnect to the backend to stream scalar or vector fields.'
						: 'This surface will populate once the engine publishes preview data.'}
					tone={!$connected ? 'warn' : 'info'}
				/>
			</div>
		{/if}

		<div id="container" class="preview-wrapper__canvas"></div>

		{#if viewMode === 'popout'}
			<div class="preview-wrapper__resize" role="presentation" onmousedown={startResize}></div>
		{/if}
	</div>
</Panel>

<style>
	.preview-mode-switcher {
		display: flex;
		gap: 0.45rem;
		flex-wrap: wrap;
	}

	.preview-toolbar {
		display: grid;
		grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr) minmax(0, 1fr) minmax(0, 1fr);
		gap: 0.8rem;
		align-items: start;
	}

	.preview-toolbar__stack {
		display: grid;
		gap: 0.8rem;
	}

	.preview-toolbar__actions {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.75rem;
	}

	.preview-wrapper {
		position: relative;
		min-height: var(--canvas-min-height);
		border-radius: var(--radius-lg);
		border: 1px solid var(--border-subtle);
		background:
			linear-gradient(180deg, rgba(6, 10, 18, 0.98), rgba(7, 10, 17, 0.98)),
			#050811;
		overflow: hidden;
	}

	.preview-wrapper__canvas {
		width: 100%;
		height: var(--canvas-min-height);
	}

	.preview-wrapper--popout {
		position: fixed;
		z-index: var(--z-popout);
		box-shadow: var(--shadow-panel);
		display: flex;
		flex-direction: column;
	}

	.preview-wrapper--popout .preview-wrapper__canvas {
		flex: 1;
		height: 100%;
	}

	.preview-wrapper--fullscreen {
		background: #04070e;
	}

	.preview-wrapper--fullscreen .preview-wrapper__canvas {
		height: 100vh;
	}

	.preview-wrapper__titlebar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		padding: 0.65rem 0.75rem;
		border-bottom: 1px solid var(--border-subtle);
		background: rgba(9, 14, 25, 0.9);
		cursor: move;
	}

	.preview-wrapper__titlebar span {
		font-size: 0.86rem;
		color: var(--text-2);
	}

	.preview-wrapper__title-actions {
		display: flex;
		gap: 0.35rem;
	}

	.preview-wrapper__empty {
		position: absolute;
		inset: 1rem;
		z-index: 2;
		display: grid;
		place-items: center;
		pointer-events: none;
	}

	.preview-wrapper__empty :global(.ui-empty) {
		pointer-events: auto;
		width: min(28rem, 100%);
		backdrop-filter: blur(12px);
	}

	.preview-wrapper__resize {
		position: absolute;
		right: 0;
		bottom: 0;
		width: 1.2rem;
		height: 1.2rem;
		cursor: nwse-resize;
		background: linear-gradient(135deg, transparent 45%, rgba(107, 167, 255, 0.45) 45%);
	}

	@media (max-width: 1279px) {
		.preview-toolbar {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 767px) {
		.preview-toolbar,
		.preview-toolbar__actions {
			grid-template-columns: 1fr;
		}
	}
</style>
