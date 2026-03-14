<script lang="ts">
	import { meshState } from '$api/incoming/mesh';
	import { previewState as p } from '$api/incoming/preview';
	import { threeDPreview, resetCamera } from './preview3D';
	import { resizeECharts } from './preview2D';
	import { onMount, onDestroy } from 'svelte';
	import { Button } from 'flowbite-svelte';
	import QuantityDropdown from './inputs/QuantityDropdown.svelte';
	import Component from './inputs/Component.svelte';
	import Layer from './inputs/Layer.svelte';
	import XDataPoints from './inputs/XDataPoints.svelte';
	import YDataPoints from './inputs/YDataPoints.svelte';
	import ResetCamera from './inputs/ResetCamera.svelte';
	import QualityPreset from './inputs/QualityPreset.svelte';
	import Toolbar3D from './inputs/Toolbar3D.svelte';
	import { get } from 'svelte/store';

	type ViewMode = 'inline' | 'popout' | 'fullscreen';
	let viewMode: ViewMode = 'inline';
	let previewWrapper: HTMLDivElement;

	// Popout drag state
	let popX = 60;
	let popY = 60;
	let popW = 700;
	let popH = 500;
	let dragging = false;
	let resizing = false;
	let dragOffX = 0;
	let dragOffY = 0;

	function setMode(mode: ViewMode) {
		// Exit fullscreen if currently in it
		if (viewMode === 'fullscreen' && document.fullscreenElement) {
			document.exitFullscreen().catch(() => {});
		}
		if (mode === 'fullscreen' && previewWrapper) {
			previewWrapper.requestFullscreen().catch(() => {});
		}
		viewMode = mode;
		scheduleResize();
	}

	function cycleMode() {
		if (viewMode === 'inline') setMode('popout');
		else if (viewMode === 'popout') setMode('fullscreen');
		else setMode('inline');
	}

	function scheduleResize() {
		setTimeout(() => {
			const container = document.getElementById('container');
			const d = get(threeDPreview);
			if (d && container) {
				d.renderer.setSize(container.clientWidth, container.clientHeight);
				d.camera.aspect = container.clientWidth / container.clientHeight;
				d.camera.updateProjectionMatrix();
			}
			resizeECharts();
		}, 100);
	}

	function onFullscreenChange() {
		if (!document.fullscreenElement && viewMode === 'fullscreen') {
			viewMode = 'inline';
		}
		scheduleResize();
	}

	// Drag handlers for popout titlebar
	function startDrag(e: MouseEvent) {
		if (resizing) return;
		dragging = true;
		dragOffX = e.clientX - popX;
		dragOffY = e.clientY - popY;
		e.preventDefault();
	}

	function startResize(e: MouseEvent) {
		resizing = true;
		dragOffX = e.clientX;
		dragOffY = e.clientY;
		e.preventDefault();
	}

	function onMouseMove(e: MouseEvent) {
		if (dragging) {
			popX = e.clientX - dragOffX;
			popY = e.clientY - dragOffY;
		} else if (resizing) {
			const dx = e.clientX - dragOffX;
			const dy = e.clientY - dragOffY;
			popW = Math.max(400, popW + dx);
			popH = Math.max(300, popH + dy);
			dragOffX = e.clientX;
			dragOffY = e.clientY;
		}
	}

	function onMouseUp() {
		if (dragging || resizing) {
			dragging = false;
			resizing = false;
			scheduleResize();
		}
	}

	onMount(() => {
		resizeECharts();
		document.addEventListener('fullscreenchange', onFullscreenChange);
		document.addEventListener('mousemove', onMouseMove);
		document.addEventListener('mouseup', onMouseUp);
	});

	onDestroy(() => {
		document.removeEventListener('fullscreenchange', onFullscreenChange);
		document.removeEventListener('mousemove', onMouseMove);
		document.removeEventListener('mouseup', onMouseUp);
	});

	$: modeLabel = viewMode === 'inline' ? '⛶' : viewMode === 'popout' ? '▢' : '✕';
	$: modeTitle = viewMode === 'inline' ? 'Pop out' : viewMode === 'popout' ? 'Fullscreen' : 'Exit';
</script>

<section>
	<h2 class="mb-4 text-2xl font-semibold">Preview</h2>

	<div class="m-1 flex flex-wrap" id="parent-fields">
		<div class="basis-1/2">
			<QuantityDropdown />
		</div>
		<div class="basis-1/2">
			<Component />
		</div>

		<div class="basis-1/2">
			{#if $p.xPossibleSizes.length > 0}
				<XDataPoints />
			{/if}
		</div>
		<div class="basis-1/2">
			{#if $p.yPossibleSizes.length > 0}
				<YDataPoints />
			{/if}
		</div>
		<div class="basis-1/4">
			<ResetCamera />
		</div>
		<div class="basis-1/4">
			<QualityPreset />
		</div>

		<div class="basis-1/2">
			<Layer />
		</div>
	</div>
	<hr />

	<div
		class="preview-wrapper relative"
		class:popout={viewMode === 'popout'}
		class:fullscreen={viewMode === 'fullscreen'}
		style={viewMode === 'popout' ? `left:${popX}px;top:${popY}px;width:${popW}px;height:${popH}px;` : ''}
		bind:this={previewWrapper}
	>
		<!-- Popout titlebar (drag handle) -->
		{#if viewMode === 'popout'}
			<!-- svelte-ignore a11y-no-static-element-interactions -->
			<div
				class="popout-titlebar flex items-center justify-between"
				on:mousedown={startDrag}
			>
				<span class="text-xs text-gray-400 select-none">3D Preview — drag to move</span>
				<div class="flex gap-1">
					<button
						class="popout-btn"
						on:click={() => setMode('inline')}
						title="Dock inline"
					>▣</button>
					<button
						class="popout-btn"
						on:click={() => setMode('fullscreen')}
						title="Fullscreen"
					>⛶</button>
					<button
						class="popout-btn"
						on:click={() => setMode('inline')}
						title="Close popout"
					>✕</button>
				</div>
			</div>
		{/if}

		<!-- Mode toggle button (inline & fullscreen) -->
		{#if viewMode !== 'popout'}
			<button
				class="absolute right-2 top-2 z-10 flex h-8 items-center gap-1 rounded-md border
					border-gray-600 bg-gray-800/90 px-3 text-sm text-gray-400
					hover:bg-gray-700 hover:text-white transition-colors backdrop-blur-sm"
				on:click={cycleMode}
				title={modeTitle}
			>
				{modeLabel} {viewMode === 'inline' ? 'Pop out' : 'Exit'}
			</button>
		{/if}

		<Toolbar3D />

		{#if $p.scalarField == null && $p.vectorFieldPositions == null}
			<div class="absolute inset-0 flex items-center justify-center text-6xl text-gray-600"
				style={viewMode === 'popout' ? 'top: 32px;' : ''}>
				NO DATA
			</div>
		{/if}
		<div id="container"></div>

		<!-- Resize handle (popout only) -->
		{#if viewMode === 'popout'}
			<!-- svelte-ignore a11y-no-static-element-interactions -->
			<div class="resize-handle" on:mousedown={startResize}></div>
		{/if}
	</div>
	<hr />
</section>

<style>
	section {
		grid-area: display;
	}
	#parent-fields > div {
		@apply p-1;
	}
	#container {
		width: 100%;
		height: 500px;
	}

	/* Popout floating window */
	.preview-wrapper.popout {
		position: fixed;
		z-index: 9999;
		background: #1a1b26;
		border: 1px solid #3b3d4a;
		border-radius: 8px;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	.preview-wrapper.popout #container {
		height: 100%;
		flex: 1;
	}
	.popout-titlebar {
		height: 32px;
		min-height: 32px;
		background: #24263a;
		border-bottom: 1px solid #3b3d4a;
		padding: 0 8px;
		cursor: move;
		user-select: none;
	}
	.popout-btn {
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		color: #888;
		font-size: 12px;
		transition: all 0.15s;
	}
	.popout-btn:hover {
		background: #3b3d4a;
		color: #fff;
	}
	.resize-handle {
		position: absolute;
		right: 0;
		bottom: 0;
		width: 16px;
		height: 16px;
		cursor: nwse-resize;
		background: linear-gradient(135deg, transparent 50%, #555 50%);
		border-radius: 0 0 8px 0;
	}

	/* Fullscreen */
	.preview-wrapper.fullscreen {
		background: #1a1b26;
	}
	.preview-wrapper.fullscreen #container {
		height: 100vh;
	}
</style>
