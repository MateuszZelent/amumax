<script lang="ts">
	import { previewState } from '$api/incoming/preview';
	import {
		threeDPreview,
		getCameraMatrix,
		setCameraViewDirection,
		orbitCamera,
		resetCamera
	} from '$lib/preview/preview3D';
	import { onMount, onDestroy } from 'svelte';

	let cubeTransform = $state('none');
	let rafId: number | null = null;
	let dragging = $state(false);
	let dragStartX = 0;
	let dragStartY = 0;
	let hasDragged = false;

	const isVisible = $derived(
		$previewState.nComp === 3 && $previewState.type === '3D' && $threeDPreview !== null
	);

	type FaceZone = { dir: [number, number, number]; type: 'face' | 'edge' | 'corner'; label?: string };
	type FaceDef = { cssTransform: string; zones: FaceZone[][] };

	const S = 30;
	const faces: FaceDef[] = [
		{ cssTransform: `translateZ(${S}px)`, zones: buildZones([0,1,0],[0,0,1],[1,0,0],'+Y') },
		{ cssTransform: `rotateY(180deg) translateZ(${S}px)`, zones: buildZones([0,-1,0],[0,0,1],[-1,0,0],'-Y') },
		{ cssTransform: `rotateY(90deg) translateZ(${S}px)`, zones: buildZones([1,0,0],[0,0,1],[0,-1,0],'+X') },
		{ cssTransform: `rotateY(-90deg) translateZ(${S}px)`, zones: buildZones([-1,0,0],[0,0,1],[0,1,0],'-X') },
		{ cssTransform: `rotateX(90deg) translateZ(${S}px)`, zones: buildZones([0,0,1],[0,1,0],[1,0,0],'+Z') },
		{ cssTransform: `rotateX(-90deg) translateZ(${S}px)`, zones: buildZones([0,0,-1],[0,-1,0],[1,0,0],'-Z') }
	];

	function buildZones(n: [number,number,number], u: [number,number,number], r: [number,number,number], label: string): FaceZone[][] {
		const add = (a: number[], b: number[], c?: number[]): [number,number,number] => {
			const res: [number,number,number] = [a[0]+b[0], a[1]+b[1], a[2]+b[2]];
			if (c) { res[0]+=c[0]; res[1]+=c[1]; res[2]+=c[2]; }
			return res;
		};
		const neg = (a: number[]): number[] => [-a[0],-a[1],-a[2]];
		return [
			[{dir:add(n,u,neg(r)),type:'corner'},{dir:add(n,u),type:'edge'},{dir:add(n,u,r),type:'corner'}],
			[{dir:add(n,neg(r)),type:'edge'},{dir:[n[0],n[1],n[2]],type:'face',label},{dir:add(n,r),type:'edge'}],
			[{dir:add(n,neg(u),neg(r)),type:'corner'},{dir:add(n,neg(u)),type:'edge'},{dir:add(n,neg(u),r),type:'corner'}]
		];
	}

	function syncLoop() {
		if ($threeDPreview) cubeTransform = getCameraMatrix();
		rafId = requestAnimationFrame(syncLoop);
	}

	function handleZoneClick(dir: [number,number,number]) {
		if (hasDragged) return;
		setCameraViewDirection(dir[0], dir[1], dir[2]);
	}

	function onPointerDown(e: PointerEvent) {
		dragging = true; hasDragged = false;
		dragStartX = e.clientX; dragStartY = e.clientY;
		(e.target as HTMLElement)?.setPointerCapture?.(e.pointerId);
		e.preventDefault();
	}

	function onPointerMove(e: PointerEvent) {
		if (!dragging) return;
		const dx = e.clientX - dragStartX, dy = e.clientY - dragStartY;
		if (Math.abs(dx) > 3 || Math.abs(dy) > 3) hasDragged = true;
		if (hasDragged) {
			orbitCamera(dx, dy);
			dragStartX = e.clientX; dragStartY = e.clientY;
		}
	}

	function onPointerUp() { dragging = false; }

	onMount(() => { syncLoop(); });
	onDestroy(() => { if (rafId !== null) cancelAnimationFrame(rafId); });
</script>

{#if isVisible}
	<!-- ViewCube: top-right -->
	<div class="vc">
		<div class="vc-scene" style="transform: {cubeTransform}"
			onpointerdown={onPointerDown} onpointermove={onPointerMove}
			onpointerup={onPointerUp} onpointerleave={onPointerUp}>
			{#each faces as face}
				<div class="vc-face" style="transform: {face.cssTransform}">
					{#each face.zones as row}
						{#each row as zone}
							<button class="vc-zone vc-zone--{zone.type}" onclick={() => handleZoneClick(zone.dir)} title={zone.label ?? ''}>
								{#if zone.label}<span class="vc-label">{zone.label}</span>{/if}
							</button>
						{/each}
					{/each}
				</div>
			{/each}
		</div>
		<button class="vc-home" onclick={resetCamera} title="Reset view">⌂</button>
	</div>

	<!-- Axis Gizmo: bottom-right -->
	<div class="ag">
		<div class="ag-scene" style="transform: {cubeTransform}">
			<!-- X axis (red) — points along +X -->
			<div class="ag-shaft ag-shaft--x" style="transform: rotateZ(-90deg) translateY(-18px)"></div>
			<div class="ag-tip ag-tip--x" style="transform: translateX(26px)"></div>
			<div class="ag-lbl ag-lbl--x" style="transform: translateX(36px)">X</div>
			<!-- Z axis (blue) — points along +Z = screen up -->
			<div class="ag-shaft ag-shaft--z" style="transform: translateY(-18px)"></div>
			<div class="ag-tip ag-tip--z" style="transform: translateY(-26px)"></div>
			<div class="ag-lbl ag-lbl--z" style="transform: translateY(-36px)">Z</div>
			<!-- Y axis (green) — points along +Y = toward viewer -->
			<div class="ag-shaft ag-shaft--y" style="transform: rotateX(90deg) translateY(-18px)"></div>
			<div class="ag-tip ag-tip--y" style="transform: translateZ(26px) rotateX(90deg)"></div>
			<div class="ag-lbl ag-lbl--y" style="transform: translateZ(36px)">Y</div>
		</div>
	</div>
{/if}

<style>
	/* ── ViewCube ─────────────────────────────── */
	.vc {
		position: absolute;
		top: 16px; right: 16px;
		width: 72px; height: 82px;
		z-index: var(--z-sticky, 10);
		perspective: 220px;
		display: flex; flex-direction: column; align-items: center;
		pointer-events: none;
	}

	.vc-scene {
		width: 60px; height: 60px;
		transform-style: preserve-3d;
		cursor: grab; touch-action: none;
		pointer-events: auto;
	}
	.vc-scene:active { cursor: grabbing; }

	.vc-face {
		position: absolute;
		width: 60px; height: 60px;
		display: grid;
		grid-template-columns: 10px 1fr 10px;
		grid-template-rows: 10px 1fr 10px;
		backface-visibility: visible;
		background: linear-gradient(135deg, rgba(15,22,40,0.8), rgba(20,30,52,0.76));
		border: 1px solid rgba(90,130,200,0.2);
	}

	.vc-zone {
		border: none; background: transparent;
		cursor: pointer; padding: 0; margin: 0;
		display: flex; align-items: center; justify-content: center;
		transition: background 0.1s;
	}
	.vc-zone:hover { background: rgba(107,167,255,0.25); }
	.vc-zone--face { color: rgba(200,215,240,0.8); }
	.vc-zone--face:hover { background: rgba(107,167,255,0.38); color: #fff; }
	.vc-zone--edge:hover { background: rgba(87,200,182,0.3); }
	.vc-zone--corner:hover { background: rgba(200,160,80,0.3); }

	.vc-label {
		font-size: 8px; font-weight: 700;
		letter-spacing: 0.05em; text-transform: uppercase;
	}

	.vc-home {
		margin-top: 3px;
		width: 20px; height: 20px; border-radius: 50%;
		background: rgba(15,22,40,0.8);
		border: 1px solid rgba(90,130,200,0.3);
		color: rgba(200,215,240,0.7);
		font-size: 12px; cursor: pointer;
		display: flex; align-items: center; justify-content: center;
		transition: all 0.15s; backdrop-filter: blur(6px); line-height: 1;
		pointer-events: auto;
	}
	.vc-home:hover { background: rgba(55,95,170,0.6); border-color: rgba(107,167,255,0.5); color: #fff; }

	/* ── Axis Gizmo ──────────────────────────── */
	.ag {
		position: absolute;
		bottom: 20px; right: 20px;
		width: 90px; height: 90px;
		z-index: var(--z-sticky, 10);
		perspective: 200px;
		pointer-events: none;
	}

	.ag-scene {
		width: 90px; height: 90px;
		transform-style: preserve-3d;
		position: relative;
	}

	/* Shared: shaft, tip, label all positioned from center */
	.ag-shaft, .ag-tip, .ag-lbl {
		position: absolute;
		left: 50%; top: 50%;
		transform-style: preserve-3d;
	}

	.ag-shaft {
		width: 2px; height: 36px;
		margin-left: -1px; margin-top: 0;
		transform-origin: center top;
		border-radius: 1px;
	}

	.ag-shaft--x { background: #e65050; }
	.ag-shaft--y { background: #50c850; }
	.ag-shaft--z { background: #5090e6; }

	.ag-tip {
		width: 0; height: 0;
		border-left: 5px solid transparent;
		border-right: 5px solid transparent;
		margin-left: -5px; margin-top: -5px;
		transform-origin: center center;
	}

	.ag-tip--x { border-bottom: 10px solid #e65050; transform: translateX(26px) rotate(-90deg); }
	.ag-tip--y { border-bottom: 10px solid #50c850; transform: translateY(-26px); }
	.ag-tip--z { border-bottom: 10px solid #5090e6; transform: translateZ(26px) rotateX(90deg); }

	.ag-lbl {
		font-size: 13px; font-weight: 800;
		pointer-events: none;
		margin-left: -5px; margin-top: -8px;
	}
	.ag-lbl--x { color: #e65050; }
	.ag-lbl--y { color: #50c850; }
	.ag-lbl--z { color: #5090e6; }
</style>
