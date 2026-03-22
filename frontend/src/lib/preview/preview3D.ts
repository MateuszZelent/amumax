import { browser } from '$app/environment';
import { previewState } from '$api/incoming/preview';
import { meshState } from '$api/incoming/mesh';
import * as THREE from 'three';
import * as BufferGeometryUtils from 'three/examples/jsm/utils/BufferGeometryUtils.js';
import { TrackballControls } from 'three/examples/jsm/controls/TrackballControls.js';
import { get, writable } from 'svelte/store';
import { disposePreview2D } from './preview2D';
import { resolveVoxelTopography } from './voxelTopography';
import { THEME } from '$lib/theme/echarts-theme';

export type QualityLevel = 'low' | 'high' | 'ultra';
export type Preview3DRenderMode = 'glyph' | 'voxel';
export type VoxelColorMode = 'orientation' | 'x' | 'y' | 'z';
export type VoxelSampling = 1 | 2 | 4;
export type TopoComponent = 'x' | 'y' | 'z';

interface QualityConfig {
	segments: number;
	useLighting: boolean;
	useHemisphere: boolean;
	antialias: boolean;
	pixelRatio: number;
}

interface ThreeDPreview {
	mesh: THREE.InstancedMesh;
	scene: THREE.Scene;
	camera: THREE.PerspectiveCamera;
	renderer: THREE.WebGLRenderer;
	controls: TrackballControls;
	rendererMode: Preview3DRenderMode;
}

const QUALITY_CONFIGS: Record<QualityLevel, QualityConfig> = {
	low: {
		segments: 6,
		useLighting: false,
		useHemisphere: false,
		antialias: false,
		pixelRatio: 1
	},
	high: {
		segments: 12,
		useLighting: true,
		useHemisphere: true,
		antialias: true,
		pixelRatio: 1
	},
	ultra: {
		segments: 16,
		useLighting: true,
		useHemisphere: true,
		antialias: true,
		pixelRatio: browser ? Math.min(window.devicePixelRatio, 2) : 1
	}
};

const STORAGE_KEYS = {
	brightness: 'preview3d_brightness',
	quality: 'preview3d_quality',
	renderMode: 'preview3d_render_mode',
	voxelOpacity: 'preview3d_voxel_opacity',
	voxelGap: 'preview3d_voxel_gap',
	voxelThreshold: 'preview3d_voxel_threshold',
	voxelColorMode: 'preview3d_voxel_color_mode',
	voxelSampling: 'preview3d_voxel_sampling',
	topoEnabled: 'preview3d_topo_enabled',
	topoComponent: 'preview3d_topo_component',
	topoMultiplier: 'preview3d_topo_multiplier'
} as const;

const COMPONENT_NEGATIVE = new THREE.Color('#2f6caa');
const COMPONENT_NEUTRAL = new THREE.Color('#f4f1ed');
const COMPONENT_POSITIVE = new THREE.Color('#cf6256');

const _dummy = new THREE.Object3D();
const _defaultUp = new THREE.Vector3(0, 1, 0);
const _tempVec = new THREE.Vector3();
const _camDir = new THREE.Vector3();
const _color = new THREE.Color();

export const brightness = writable<number>(loadBrightness());
export const qualityLevel = writable<QualityLevel>(loadQuality());
export const renderMode = writable<Preview3DRenderMode>(loadRenderMode());
export const voxelOpacity = writable<number>(
	loadClampedNumber(STORAGE_KEYS.voxelOpacity, 0.5, 0.15, 0.95)
);
export const voxelGap = writable<number>(
	loadClampedNumber(STORAGE_KEYS.voxelGap, 0.14, 0.02, 0.42)
);
export const voxelThreshold = writable<number>(
	loadClampedNumber(STORAGE_KEYS.voxelThreshold, 0.08, 0, 0.95)
);
export const voxelColorMode = writable<VoxelColorMode>(loadVoxelColorMode());
export const voxelSampling = writable<VoxelSampling>(loadVoxelSampling());
export const threeDPreview = writable<ThreeDPreview | null>(null);
export const visibleRenderCount = writable<number>(0);
export const topoEnabled = writable<boolean>(loadTopoEnabled());
export const topoComponent = writable<TopoComponent>(loadTopoComponent());
export const topoMultiplier = writable<number>(
	loadClampedNumber(STORAGE_KEYS.topoMultiplier, 5, 0.5, 50)
);

let animationFrameId: number | null = null;
let resizeObserver: ResizeObserver | null = null;

function loadClampedNumber(key: string, fallback: number, min: number, max: number) {
	if (!browser) {
		return fallback;
	}

	const raw = Number.parseFloat(window.localStorage.getItem(key) || '');
	if (!Number.isFinite(raw)) {
		return fallback;
	}

	return Math.max(min, Math.min(max, raw));
}

function loadBrightness() {
	return loadClampedNumber(STORAGE_KEYS.brightness, 1.5, 0.3, 3.0);
}

function loadQuality(): QualityLevel {
	if (!browser) {
		return 'high';
	}

	const stored = window.localStorage.getItem(STORAGE_KEYS.quality);
	if (stored && stored in QUALITY_CONFIGS) {
		return stored as QualityLevel;
	}

	return 'high';
}

function loadRenderMode(): Preview3DRenderMode {
	if (!browser) {
		return 'glyph';
	}

	return window.localStorage.getItem(STORAGE_KEYS.renderMode) === 'voxel' ? 'voxel' : 'glyph';
}

function loadVoxelColorMode(): VoxelColorMode {
	if (!browser) {
		return 'orientation';
	}

	const stored = window.localStorage.getItem(STORAGE_KEYS.voxelColorMode);
	if (stored === 'x' || stored === 'y' || stored === 'z') {
		return stored;
	}

	return 'orientation';
}

function loadVoxelSampling(): VoxelSampling {
	if (!browser) {
		return 1;
	}

	const stored = Number.parseInt(window.localStorage.getItem(STORAGE_KEYS.voxelSampling) || '', 10);
	return stored === 2 || stored === 4 ? stored : 1;
}

function loadTopoEnabled(): boolean {
	if (!browser) return false;
	return window.localStorage.getItem(STORAGE_KEYS.topoEnabled) === 'true';
}

function loadTopoComponent(): TopoComponent {
	if (!browser) return 'z';
	const stored = window.localStorage.getItem(STORAGE_KEYS.topoComponent);
	if (stored === 'x' || stored === 'y' || stored === 'z') return stored;
	return 'z';
}

function persistSetting(key: string, value: string | number) {
	if (!browser) {
		return;
	}

	window.localStorage.setItem(key, String(value));
}

function has3DPreviewData() {
	const state = get(previewState);
	return state.type === '3D' && state.nComp === 3;
}

function rebuildPreview3D() {
	if (!has3DPreviewData()) {
		return;
	}

	disposePreview3D();
	init();
}

function getConfig(): QualityConfig {
	return QUALITY_CONFIGS[get(qualityLevel)];
}

function getDepthCells() {
	return get(previewState).allLayers ? Math.max(get(meshState).Nz, 1) : 1;
}

function getPreviewWidthCells() {
	const state = get(previewState);
	return Math.max(state.appliedXChosenSize || state.xChosenSize, 1);
}

function getPreviewHeightCells() {
	const state = get(previewState);
	return Math.max(state.appliedYChosenSize || state.yChosenSize, 1);
}

function componentValue(
	vector: { x: number; y: number; z: number },
	mode: Exclude<VoxelColorMode, 'orientation'>
) {
	switch (mode) {
		case 'x':
			return vector.x;
		case 'y':
			return vector.y;
		case 'z':
			return vector.z;
	}
}

function vectorMagnitude(vector: { x: number; y: number; z: number }) {
	return Math.sqrt(vector.x * vector.x + vector.y * vector.y + vector.z * vector.z);
}

function magnetizationHSL(vx: number, vy: number, vz: number, color: THREE.Color) {
	const hue = Math.atan2(vy, vx) / (Math.PI * 2);
	const saturation = Math.min(1, Math.sqrt(vx * vx + vy * vy));
	const lightness = THREE.MathUtils.clamp((vz + 1) / 2, 0.18, 0.84);
	color.setHSL((hue + 1) % 1, saturation, lightness);
}

function applyComponentColor(value: number, color: THREE.Color) {
	const normalized = THREE.MathUtils.clamp(value, -1, 1);

	if (normalized < 0) {
		color.copy(COMPONENT_NEUTRAL).lerp(COMPONENT_NEGATIVE, Math.abs(normalized));
		return;
	}

	color.copy(COMPONENT_NEUTRAL).lerp(COMPONENT_POSITIVE, normalized);
}

function applyVoxelColor(vector: { x: number; y: number; z: number }, color: THREE.Color) {
	const mode = get(voxelColorMode);

	if (mode === 'orientation') {
		magnetizationHSL(vector.x, vector.y, vector.z, color);
		return;
	}

	applyComponentColor(componentValue(vector, mode), color);
}

function createArrowGeometry() {
	const seg = getConfig().segments;
	const shaftGeometry = new THREE.CylinderGeometry(0.05, 0.05, 0.55, seg);
	shaftGeometry.translate(0, -0.06, 0);
	const headGeometry = new THREE.ConeGeometry(0.2, 0.4, seg);
	headGeometry.translate(0, 0.4, 0);
	const arrowGeometry = BufferGeometryUtils.mergeGeometries([shaftGeometry, headGeometry]);

	if (!arrowGeometry) {
		throw new Error('Could not create arrow geometry');
	}

	arrowGeometry.computeVertexNormals();
	return arrowGeometry;
}

function createVoxelGeometry() {
	return new THREE.BoxGeometry(1, 1, 1);
}

function createMaterial(mode: Preview3DRenderMode): THREE.Material {
	const cfg = getConfig();

	if (mode === 'voxel') {
		if (cfg.useLighting) {
			return new THREE.MeshPhongMaterial({
				transparent: true,
				opacity: get(voxelOpacity),
				depthWrite: false,
				shininess: 24,
				specular: new THREE.Color(0x24334c)
			});
		}

		return new THREE.MeshBasicMaterial({
			transparent: true,
			opacity: get(voxelOpacity),
			depthWrite: false
		});
	}

	if (cfg.useLighting) {
		return new THREE.MeshPhongMaterial({
			shininess: 60,
			specular: new THREE.Color(0x444444)
		});
	}

	return new THREE.MeshBasicMaterial();
}

function createMesh(): THREE.InstancedMesh {
	const mode = get(renderMode);
	const geometry = mode === 'voxel' ? createVoxelGeometry() : createArrowGeometry();
	const material = createMaterial(mode);
	const count = get(previewState).vectorFieldValues.length;
	const mesh = new THREE.InstancedMesh(geometry, material, count);

	mesh.frustumCulled = false;
	mesh.renderOrder = mode === 'voxel' ? 2 : 1;
	mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);
	mesh.instanceColor = new THREE.InstancedBufferAttribute(
		new Float32Array(Math.max(count, 1) * 3),
		3
	);

	return mesh;
}

function createCamera() {
	const container = document.getElementById('container');
	const width = container?.offsetWidth || 1;
	const height = container?.offsetHeight || 1;
	const camera = new THREE.PerspectiveCamera(50, width / height, 0.1, 1000);
	const depthCells = getDepthCells();
	const xSize = getPreviewWidthCells();
	const ySize = getPreviewHeightCells();
	camera.position.set(xSize / 2, ySize / 2, Math.max(xSize, ySize, depthCells) * 1.5);
	return camera;
}

function createRenderer() {
	const cfg = getConfig();
	const renderer = new THREE.WebGLRenderer({
		antialias: cfg.antialias,
		alpha: false
	});

	renderer.setPixelRatio(cfg.pixelRatio);
	renderer.setClearColor(THEME.bg, 1);
	renderer.toneMapping = THREE.ACESFilmicToneMapping;
	renderer.toneMappingExposure = 1.05;

	const container = document.getElementById('container');
	if (!container) {
		throw new Error('Container not found');
	}

	renderer.setSize(container.clientWidth, container.clientHeight);
	container.appendChild(renderer.domElement);
	return renderer;
}

function createControls(camera: THREE.PerspectiveCamera, renderer: THREE.WebGLRenderer) {
	const controls = new TrackballControls(camera, renderer.domElement);
	const depthCells = getDepthCells();

	controls.dynamicDampingFactor = 1;
	controls.panSpeed = 0.8;
	controls.rotateSpeed = 1;
	controls.target.set(getPreviewWidthCells() / 2, getPreviewHeightCells() / 2, depthCells / 2);
	controls.update();

	return controls;
}

function createScene() {
	const cfg = getConfig();
	const scene = new THREE.Scene();
	scene.background = new THREE.Color(THEME.bg);

	if (!cfg.useLighting) {
		return scene;
	}

	const brightnessValue = get(brightness);

	const dirLight = new THREE.DirectionalLight(0xffffff, 1.8 * brightnessValue);
	dirLight.position.set(1, 2, 3);
	dirLight.userData.baseIntensity = 1.8;
	scene.add(dirLight);

	const fillLight = new THREE.DirectionalLight(0xccccff, 0.8 * brightnessValue);
	fillLight.position.set(-2, 0, 1);
	fillLight.userData.baseIntensity = 0.8;
	scene.add(fillLight);

	const backLight = new THREE.DirectionalLight(0xffffff, 0.5 * brightnessValue);
	backLight.position.set(0, -1, -2);
	backLight.userData.baseIntensity = 0.5;
	scene.add(backLight);

	const ambient = new THREE.AmbientLight(0x8888aa, 1.0 * brightnessValue);
	ambient.userData.baseIntensity = 1.0;
	scene.add(ambient);

	if (cfg.useHemisphere) {
		const hemiLight = new THREE.HemisphereLight(0x8898bf, 0x293245, 0.6 * brightnessValue);
		hemiLight.userData.baseIntensity = 0.6;
		scene.add(hemiLight);
	}

	return scene;
}

function updateSceneLights() {
	const display = get(threeDPreview);
	if (!display) {
		return;
	}

	const brightnessValue = get(brightness);
	display.scene.traverse((child) => {
		if (
			child instanceof THREE.DirectionalLight ||
			child instanceof THREE.AmbientLight ||
			child instanceof THREE.HemisphereLight
		) {
			(child as THREE.Light).intensity = (child.userData.baseIntensity || 1) * brightnessValue;
		}
	});
}

function updateMaterialAppearance() {
	const display = get(threeDPreview);
	if (!display || Array.isArray(display.mesh.material) || display.rendererMode !== 'voxel') {
		return;
	}

	display.mesh.material.transparent = true;
	display.mesh.material.opacity = get(voxelOpacity);
	display.mesh.material.depthWrite = false;
	display.mesh.material.needsUpdate = true;
}

function isSampledPosition(position: { x: number; y: number; z: number }) {
	const step = get(voxelSampling);
	if (step === 1) {
		return true;
	}

	if (position.x % step !== 0 || position.y % step !== 0) {
		return false;
	}

	if (get(previewState).allLayers && position.z % step !== 0) {
		return false;
	}

	return true;
}

function updateGlyphMesh(mesh: THREE.InstancedMesh) {
	const values = get(previewState).vectorFieldValues;
	const positions = get(previewState).vectorFieldPositions;
	const count = Math.min(values.length, positions.length, mesh.count);
	const instanceColor = mesh.instanceColor;

	if (!instanceColor) {
		visibleRenderCount.set(0);
		return;
	}

	const colors = instanceColor.array as Float32Array;
	let visibleCount = 0;

	for (let i = 0; i < count; i++) {
		const vector = values[i];
		const position = positions[i];
		const isVisible = vector.x !== 0 || vector.y !== 0 || vector.z !== 0;

		_dummy.position.set(position.x, position.y, position.z);

		if (!isVisible) {
			_dummy.scale.set(0, 0, 0);
			_dummy.quaternion.identity();
		} else {
			visibleCount += 1;
			_dummy.scale.set(1, 1, 1);
			_tempVec.set(vector.x, vector.y, vector.z).normalize();
			_dummy.quaternion.setFromUnitVectors(_defaultUp, _tempVec);
		}

		magnetizationHSL(vector.x, vector.y, vector.z, _color);
		colors[i * 3 + 0] = _color.r;
		colors[i * 3 + 1] = _color.g;
		colors[i * 3 + 2] = _color.b;

		_dummy.updateMatrix();
		mesh.setMatrixAt(i, _dummy.matrix);
	}

	visibleRenderCount.set(visibleCount);
	mesh.instanceMatrix.needsUpdate = true;
	instanceColor.needsUpdate = true;
}

function updateVoxelMesh(mesh: THREE.InstancedMesh) {
	const values = get(previewState).vectorFieldValues;
	const positions = get(previewState).vectorFieldPositions;
	const count = Math.min(values.length, positions.length, mesh.count);
	const instanceColor = mesh.instanceColor;

	if (!instanceColor) {
		visibleRenderCount.set(0);
		return;
	}

	const colors = instanceColor.array as Float32Array;
	const step = get(voxelSampling);
	const baseScale = Math.max(0.12, step * (1 - get(voxelGap)));
	const depthScale = get(previewState).allLayers ? baseScale : Math.max(0.22, baseScale * 0.42);
	const threshold = get(voxelThreshold);
	const colorMode = get(voxelColorMode);
	const topo = get(topoEnabled);
	const topoComp = get(topoComponent);
	const topoMul = get(topoMultiplier);
	let visibleCount = 0;

	for (let i = 0; i < count; i++) {
		const vector = values[i];
		const position = positions[i];
		const metric =
			colorMode === 'orientation'
				? vectorMagnitude(vector)
				: Math.abs(componentValue(vector, colorMode));
		const isVisible = isSampledPosition(position) && metric >= threshold;

		let pz = position.z;
		let voxelDepth = depthScale;
		if (topo) {
			const topoDisplacement = componentValue(vector, topoComp) * topoMul;
			const topography = resolveVoxelTopography(position.z, depthScale, topoDisplacement);
			pz = topography.centerZ;
			voxelDepth = topography.depthScale;
		}

		_dummy.position.set(position.x, position.y, pz);
		_dummy.quaternion.identity();

		if (!isVisible) {
			_dummy.scale.set(0, 0, 0);
		} else {
			visibleCount += 1;
			_dummy.scale.set(baseScale, baseScale, voxelDepth);
		}

		applyVoxelColor(vector, _color);
		colors[i * 3 + 0] = _color.r;
		colors[i * 3 + 1] = _color.g;
		colors[i * 3 + 2] = _color.b;

		_dummy.updateMatrix();
		mesh.setMatrixAt(i, _dummy.matrix);
	}

	visibleRenderCount.set(visibleCount);
	mesh.instanceMatrix.needsUpdate = true;
	instanceColor.needsUpdate = true;
	updateMaterialAppearance();
}

function init() {
	const scene = createScene();
	const camera = createCamera();
	const renderer = createRenderer();
	const controls = createControls(camera, renderer);
	const mesh = createMesh();

	scene.add(mesh);

	threeDPreview.set({
		mesh,
		scene,
		camera,
		renderer,
		controls,
		rendererMode: get(renderMode)
	});

	update();

	const container = document.getElementById('container');
	if (container) {
		if (!resizeObserver) {
			resizeObserver = new ResizeObserver(() => {
				const current = document.getElementById('container');
				if (!current) {
					return;
				}

				const width = current.clientWidth;
				const height = current.clientHeight;
				renderer.setSize(width, height);
				camera.aspect = width / height;
				camera.updateProjectionMatrix();
			});
		}

		resizeObserver.disconnect();
		resizeObserver.observe(container);
	}

	function animate() {
		animationFrameId = requestAnimationFrame(animate);
		controls.update();
		renderer.render(scene, camera);
	}

	animate();
}

function update() {
	const display = get(threeDPreview);
	if (!display) {
		return;
	}

	if (display.rendererMode !== get(renderMode)) {
		rebuildPreview3D();
		return;
	}

	if (display.rendererMode === 'voxel') {
		updateVoxelMesh(display.mesh);
		return;
	}

	updateGlyphMesh(display.mesh);
}

export function preview3D() {
	if (!has3DPreviewData()) {
		disposePreview3D();
		return;
	}

	if (get(previewState).refresh) {
		disposePreview2D();
		disposePreview3D();
		init();
		return;
	}

	update();
}

export function disposePreview3D() {
	const container = document.getElementById('container');
	const display = get(threeDPreview);
	visibleRenderCount.set(0);

	if (animationFrameId !== null) {
		cancelAnimationFrame(animationFrameId);
		animationFrameId = null;
	}

	if (display) {
		display.controls.dispose();
		display.renderer.dispose();
		display.scene.traverse((child) => {
			if (child instanceof THREE.Mesh || child instanceof THREE.InstancedMesh) {
				child.geometry.dispose();
				if (Array.isArray(child.material)) {
					child.material.forEach((material) => material.dispose());
				} else if (child.material) {
					child.material.dispose();
				}
			}
		});
		threeDPreview.set(null);

		if (container) {
			container.innerHTML = '';
		}
	}

	if (resizeObserver) {
		resizeObserver.disconnect();
		resizeObserver = null;
	}
}

export function resizeECharts() {
	const container = document.getElementById('container');
	if (!container) {
		return;
	}

	if (!resizeObserver) {
		resizeObserver = new ResizeObserver(() => {
			const display = get(threeDPreview);
			if (!display) {
				return;
			}

			display.renderer.setSize(container.clientWidth, container.clientHeight);
			display.camera.aspect = container.clientWidth / container.clientHeight;
			display.camera.updateProjectionMatrix();
		});
	}

	resizeObserver.disconnect();
	resizeObserver.observe(container);

	const display = get(threeDPreview);
	if (display) {
		display.renderer.setSize(container.clientWidth, container.clientHeight);
		display.camera.aspect = container.clientWidth / container.clientHeight;
		display.camera.updateProjectionMatrix();
	}
}

export function setBrightness(value: number) {
	const clamped = Math.max(0.3, Math.min(3.0, value));
	persistSetting(STORAGE_KEYS.brightness, clamped);
	brightness.set(clamped);
	updateSceneLights();
}

export function setQuality(level: QualityLevel) {
	persistSetting(STORAGE_KEYS.quality, level);
	qualityLevel.set(level);
	rebuildPreview3D();
}

export function setRenderMode(mode: Preview3DRenderMode) {
	persistSetting(STORAGE_KEYS.renderMode, mode);
	renderMode.set(mode);
	rebuildPreview3D();
}

export function setVoxelOpacity(value: number) {
	const clamped = Math.max(0.15, Math.min(0.95, value));
	persistSetting(STORAGE_KEYS.voxelOpacity, clamped);
	voxelOpacity.set(clamped);
	updateMaterialAppearance();
}

export function setVoxelGap(value: number) {
	const clamped = Math.max(0.02, Math.min(0.42, value));
	persistSetting(STORAGE_KEYS.voxelGap, clamped);
	voxelGap.set(clamped);
	if (get(renderMode) === 'voxel') {
		update();
	}
}

export function setVoxelThreshold(value: number) {
	const clamped = Math.max(0, Math.min(0.95, value));
	persistSetting(STORAGE_KEYS.voxelThreshold, clamped);
	voxelThreshold.set(clamped);
	if (get(renderMode) === 'voxel') {
		update();
	}
}

export function setVoxelColorMode(mode: VoxelColorMode) {
	persistSetting(STORAGE_KEYS.voxelColorMode, mode);
	voxelColorMode.set(mode);
	if (get(renderMode) === 'voxel') {
		update();
	}
}

export function setVoxelSampling(value: VoxelSampling) {
	persistSetting(STORAGE_KEYS.voxelSampling, value);
	voxelSampling.set(value);
	if (get(renderMode) === 'voxel') {
		update();
	}
}

export function setTopoEnabled(value: boolean) {
	persistSetting(STORAGE_KEYS.topoEnabled, String(value));
	topoEnabled.set(value);
	if (get(renderMode) === 'voxel') {
		update();
	}
}

export function setTopoComponent(comp: TopoComponent) {
	persistSetting(STORAGE_KEYS.topoComponent, comp);
	topoComponent.set(comp);
	if (get(renderMode) === 'voxel' && get(topoEnabled)) {
		update();
	}
}

export function setTopoMultiplier(value: number) {
	const clamped = Math.max(0.5, Math.min(50, value));
	persistSetting(STORAGE_KEYS.topoMultiplier, clamped);
	topoMultiplier.set(clamped);
	if (get(renderMode) === 'voxel' && get(topoEnabled)) {
		update();
	}
}

export function resetCamera() {
	const depthCells = getDepthCells();
	const xSize = getPreviewWidthCells();
	const ySize = getPreviewHeightCells();
	const positionZ = Math.max(xSize, ySize, depthCells) * 1.5;
	const display = get(threeDPreview);

	if (!display) {
		return;
	}

	display.camera.position.set(xSize / 2, ySize / 2, positionZ);
	display.camera.up.set(0, 1, 0);
	display.camera.lookAt(xSize / 2, ySize / 2, depthCells / 2);
	display.controls.target.set(xSize / 2, ySize / 2, depthCells / 2);
	display.controls.update();
}

export function setCameraViewDirection(dx: number, dy: number, dz: number) {
	const display = get(threeDPreview);
	if (!display) return;

	const len = Math.sqrt(dx * dx + dy * dy + dz * dz);
	if (len === 0) return;
	dx /= len;
	dy /= len;
	dz /= len;

	const depthCells = getDepthCells();
	const xSize = getPreviewWidthCells();
	const ySize = getPreviewHeightCells();
	const dist = Math.max(xSize, ySize, depthCells) * 1.5;

	const cx = xSize / 2,
		cy = ySize / 2,
		cz = depthCells / 2;

	let ux = 0,
		uy = 1,
		uz = 0;
	if (Math.abs(dy) > 0.9) {
		ux = 0;
		uy = 0;
		uz = dy > 0 ? -1 : 1;
	}

	display.camera.position.set(cx + dx * dist, cy + dy * dist, cz + dz * dist);
	display.camera.up.set(ux, uy, uz);
	display.camera.lookAt(cx, cy, cz);
	display.controls.target.set(cx, cy, cz);
	display.controls.update();
}

export function orbitCamera(deltaX: number, deltaY: number) {
	const display = get(threeDPreview);
	if (!display) return;

	const camera = display.camera;
	const target = display.controls.target;
	const offset = new THREE.Vector3().subVectors(camera.position, target);
	const radius = offset.length();

	let theta = Math.atan2(offset.x, offset.z);
	let phi = Math.asin(THREE.MathUtils.clamp(offset.y / radius, -1, 1));

	theta -= deltaX * 0.01;
	phi += deltaY * 0.01;
	phi = THREE.MathUtils.clamp(phi, -Math.PI / 2 + 0.05, Math.PI / 2 - 0.05);

	offset.x = radius * Math.cos(phi) * Math.sin(theta);
	offset.y = radius * Math.sin(phi);
	offset.z = radius * Math.cos(phi) * Math.cos(theta);

	camera.position.copy(target).add(offset);
	camera.up.set(0, 1, 0);
	camera.lookAt(target);
	display.controls.update();
}

export function getCameraMatrix(): string {
	const display = get(threeDPreview);
	if (!display) return 'none';

	const cam = display.camera;
	const target = display.controls.target;

	// Direction from target toward camera (NOT camera toward target)
	const dir = _camDir.subVectors(cam.position, target).normalize();

	// Spherical angles: theta = horizontal (around Y), phi = vertical
	const theta = Math.atan2(dir.x, dir.z); // 0 when camera at +Z (front)
	const phi = Math.asin(THREE.MathUtils.clamp(dir.y, -1, 1));

	// CSS rotations: rotate the cube so the face the camera sees is toward the viewer
	// rotateY(-theta): horizontal orbit
	// rotateX(phi): vertical tilt (positive phi = camera above = tilt cube forward)
	return `rotateX(${phi}rad) rotateY(${-theta}rad)`;
}
