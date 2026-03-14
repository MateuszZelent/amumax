import { previewState } from '$api/incoming/preview';
import { meshState } from '$api/incoming/mesh';
import * as THREE from 'three';
import * as BufferGeometryUtils from 'three/examples/jsm/utils/BufferGeometryUtils.js';
import { TrackballControls } from 'three/examples/jsm/controls/TrackballControls.js';
import { get } from 'svelte/store';
import { writable } from 'svelte/store';
import { disposePreview2D } from './preview2D';

// ─── Quality presets ────────────────────────────────────────────────
export type QualityLevel = 'low' | 'high' | 'ultra';

interface QualityConfig {
	segments: number;
	useLighting: boolean;
	useHemisphere: boolean;
	antialias: boolean;
	pixelRatio: number;
}

const QUALITY_CONFIGS: Record<QualityLevel, QualityConfig> = {
	low: {
		segments: 6,
		useLighting: false,
		useHemisphere: false,
		antialias: false,
		pixelRatio: 1,
	},
	high: {
		segments: 12,
		useLighting: true,
		useHemisphere: true,
		antialias: true,
		pixelRatio: 1,
	},
	ultra: {
		segments: 16,
		useLighting: true,
		useHemisphere: true,
		antialias: true,
		pixelRatio: Math.min(window.devicePixelRatio, 2),
	},
};

// ─── Brightness control ─────────────────────────────────────────────
function loadBrightness(): number {
	const v = parseFloat(localStorage.getItem('preview3d_brightness') || '');
	return isNaN(v) ? 1.5 : Math.max(0.3, Math.min(3.0, v));
}

export const brightness = writable<number>(loadBrightness());

export function setBrightness(val: number) {
	const clamped = Math.max(0.3, Math.min(3.0, val));
	localStorage.setItem('preview3d_brightness', String(clamped));
	brightness.set(clamped);
	updateSceneLights();
}

function updateSceneLights() {
	const d = get(threeDPreview);
	if (!d) return;
	const b = get(brightness);
	d.scene.traverse((child) => {
		if (child instanceof THREE.DirectionalLight ||
			child instanceof THREE.AmbientLight ||
			child instanceof THREE.HemisphereLight) {
			(child as any).intensity = (child.userData.baseIntensity || 1) * b;
		}
	});
}

function loadQuality(): QualityLevel {
	const stored = localStorage.getItem('preview3d_quality');
	if (stored && stored in QUALITY_CONFIGS) return stored as QualityLevel;
	return 'high';
}

export const qualityLevel = writable<QualityLevel>(loadQuality());

export function setQuality(level: QualityLevel) {
	localStorage.setItem('preview3d_quality', level);
	qualityLevel.set(level);
	const ps = get(previewState);
	if (ps.vectorFieldPositions != null) {
		disposePreview3D();
		init();
	}
}

function getConfig(): QualityConfig {
	return QUALITY_CONFIGS[get(qualityLevel)];
}

// ─── Shared reusable objects (avoid per-frame allocations) ──────────
const _dummy = new THREE.Object3D();
const _defaultUp = new THREE.Vector3(0, 1, 0);
const _tempVec = new THREE.Vector3();
const _color = new THREE.Color();

// ─── HSL color from magnetization vector ────────────────────────────
function magnetizationHSL(vx: number, vy: number, vz: number, color: THREE.Color): void {
	const h = Math.atan2(vy, vx) / Math.PI / 2;
	const s = Math.sqrt(vx * vx + vy * vy + vz * vz);
	const l = (vz + 1) / 2;
	color.setHSL(h, s, l);
}

// ─── Main API ───────────────────────────────────────────────────────
export function preview3D() {
	if (get(previewState).vectorFieldPositions == null) {
		disposePreview3D();
		return;
	} else if (get(previewState).refresh) {
		disposePreview2D();
		disposePreview3D();
		init();
	} else {
		update();
	}
}

export function disposePreview3D() {
	const container = document.getElementById('container');
	const displayInstance = get(threeDPreview);
	if (animationFrameId !== null) {
		cancelAnimationFrame(animationFrameId);
		animationFrameId = null;
	}
	if (displayInstance) {
		// [Fix #13] Dispose controls to remove DOM event listeners
		displayInstance.controls.dispose();

		displayInstance.renderer.dispose();
		displayInstance.scene.traverse((child) => {
			if (child instanceof THREE.Mesh || child instanceof THREE.InstancedMesh) {
				child.geometry.dispose();
				if (Array.isArray(child.material)) {
					child.material.forEach((m) => m.dispose());
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
}

// ─── Types & state ──────────────────────────────────────────────────
interface ThreeDPreview {
	mesh: THREE.InstancedMesh;
	scene: THREE.Scene;
	camera: THREE.PerspectiveCamera;
	renderer: THREE.WebGLRenderer;
	controls: TrackballControls;
	isInitialized: boolean;
}
export const threeDPreview = writable<ThreeDPreview | null>(null);
let animationFrameId: number | null = null;

// ─── Geometry & Material ────────────────────────────────────────────
function createMesh(): THREE.InstancedMesh {
	const cfg = getConfig();
	const seg = cfg.segments;

	const shaftGeometry = new THREE.CylinderGeometry(0.05, 0.05, 0.55, seg);
	shaftGeometry.translate(0, -0.06, 0);
	const headGeometry = new THREE.ConeGeometry(0.2, 0.4, seg);
	headGeometry.translate(0, 0.4, 0);
	const arrowGeometry = BufferGeometryUtils.mergeGeometries([shaftGeometry, headGeometry]);

	arrowGeometry.computeVertexNormals();

	let arrowMaterial: THREE.Material;
	if (cfg.useLighting) {
		arrowMaterial = new THREE.MeshPhongMaterial({
			vertexColors: false,
			shininess: 60,
			specular: new THREE.Color(0x444444),
		});
	} else {
		arrowMaterial = new THREE.MeshBasicMaterial({ vertexColors: false });
	}

	const count = get(previewState).vectorFieldValues.length;
	const mesh = new THREE.InstancedMesh(arrowGeometry, arrowMaterial, count);

	// [Fix #11] Disable per-instance frustum culling for dense fields
	mesh.frustumCulled = false;

	return mesh;
}

// ─── Camera ─────────────────────────────────────────────────────────
function createCamera(): THREE.PerspectiveCamera {
	const div = document.getElementById('container');
	const width = div?.offsetWidth || 1;
	const height = div?.offsetHeight || 1;
	const camera = new THREE.PerspectiveCamera(50, width / height, 0.1, 1000);

	const dims = [get(previewState).xChosenSize, get(previewState).yChosenSize];
	const nz = get(previewState).allLayers ? get(meshState).Nz : 1;
	camera.position.set(dims[0] / 2, dims[1] / 2, Math.max(dims[0], dims[1], nz) * 1.5);
	return camera;
}

// ─── Renderer ───────────────────────────────────────────────────────
function createRenderer() {
	const cfg = getConfig();
	const renderer = new THREE.WebGLRenderer({
		antialias: cfg.antialias,
		alpha: false,
	});
	renderer.setPixelRatio(cfg.pixelRatio);

	const container = document.getElementById('container');
	if (!container) throw new Error('Container not found');
	renderer.setSize(container.clientWidth, container.clientHeight);
	container.appendChild(renderer.domElement);
	return renderer;
}

// ─── Controls ───────────────────────────────────────────────────────
function createControls(camera: THREE.PerspectiveCamera, renderer: THREE.WebGLRenderer) {
	const controls = new TrackballControls(camera, renderer.domElement);
	controls.dynamicDampingFactor = 1;
	controls.panSpeed = 0.8;
	controls.rotateSpeed = 1;
	const nz = get(previewState).allLayers ? get(meshState).Nz : 1;
	controls.target.set(get(previewState).xChosenSize / 2, get(previewState).yChosenSize / 2, nz / 2);
	controls.update();
	return controls;
}

// ─── Scene & Lighting ───────────────────────────────────────────────
function createScene(): THREE.Scene {
	const cfg = getConfig();
	const scene = new THREE.Scene();
	scene.background = new THREE.Color(0x1a1b26);

	if (cfg.useLighting) {
		const b = get(brightness);

		const dirLight = new THREE.DirectionalLight(0xffffff, 1.8 * b);
		dirLight.position.set(1, 2, 3);
		dirLight.userData.baseIntensity = 1.8;
		scene.add(dirLight);

		const fillLight = new THREE.DirectionalLight(0xccccff, 0.8 * b);
		fillLight.position.set(-2, 0, 1);
		fillLight.userData.baseIntensity = 0.8;
		scene.add(fillLight);

		const backLight = new THREE.DirectionalLight(0xffffff, 0.5 * b);
		backLight.position.set(0, -1, -2);
		backLight.userData.baseIntensity = 0.5;
		scene.add(backLight);

		const ambient = new THREE.AmbientLight(0x8888aa, 1.0 * b);
		ambient.userData.baseIntensity = 1.0;
		scene.add(ambient);

		if (cfg.useHemisphere) {
			const hemiLight = new THREE.HemisphereLight(0x8888aa, 0x444466, 0.6 * b);
			hemiLight.userData.baseIntensity = 0.6;
			scene.add(hemiLight);
		}
	}

	return scene;
}

// ─── Arrow coloring & positioning ───────────────────────────────────
function addArrowsToMesh(mesh: THREE.InstancedMesh) {
	const vectorFieldValues = get(previewState).vectorFieldValues;
	const vectorFieldPositions = get(previewState).vectorFieldPositions;

	const instanceColorLength = vectorFieldPositions.length * 3;
	const instanceColor = new THREE.InstancedBufferAttribute(new Float32Array(instanceColorLength), 3);
	const colors = instanceColor.array;

	for (let i = 0; i < vectorFieldValues.length; i++) {
		const val = vectorFieldValues[i];
		const pos = vectorFieldPositions[i];

		// [Fix #5] Scale zero-vectors to zero so they're invisible
		if (val.x === 0 && val.y === 0 && val.z === 0) {
			_dummy.position.set(pos.x, pos.y, pos.z);
			_dummy.scale.set(0, 0, 0);
		} else {
			_dummy.position.set(pos.x, pos.y, pos.z);
			_dummy.scale.set(1, 1, 1);
		}

		magnetizationHSL(val.x, val.y, val.z, _color);
		colors[i * 3 + 0] = _color.r;
		colors[i * 3 + 1] = _color.g;
		colors[i * 3 + 2] = _color.b;

		_dummy.updateMatrix();
		mesh.setMatrixAt(i, _dummy.matrix);
	}

	mesh.instanceMatrix.needsUpdate = true;
	instanceColor.needsUpdate = true;
	mesh.instanceColor = instanceColor;
}

// ─── Init ───────────────────────────────────────────────────────────
function init() {
	const scene = createScene();
	const camera = createCamera();
	const renderer = createRenderer();
	const controls = createControls(camera, renderer);
	const mesh = createMesh();
	addArrowsToMesh(mesh);
	scene.add(mesh);

	threeDPreview.set({
		mesh,
		scene,
		camera,
		renderer,
		controls,
		isInitialized: true,
	});

	// [Fix #2] update() after store is set so it can find the mesh
	update();

	// [Fix #12] Handle window resize for 3D renderer
	const onResize = () => {
		const container = document.getElementById('container');
		if (!container) return;
		const w = container.clientWidth;
		const h = container.clientHeight;
		renderer.setSize(w, h);
		camera.aspect = w / h;
		camera.updateProjectionMatrix();
	};
	window.addEventListener('resize', onResize);

	function animate() {
		animationFrameId = requestAnimationFrame(animate);
		controls.update();
		renderer.render(scene, camera);
	}
	animate();
}

// ─── Update ─────────────────────────────────────────────────────────
function update() {
	const d = get(threeDPreview);
	if (!d) return;

	const mesh = d.mesh;
	const vectorField = get(previewState).vectorFieldValues;
	const instanceColor = mesh.instanceColor;
	if (!instanceColor) return;
	const colors = instanceColor.array;

	for (let i = 0; i < vectorField.length; i++) {
		const vector = vectorField[i];

		mesh.getMatrixAt(i, _dummy.matrix);
		_dummy.matrix.decompose(_dummy.position, _dummy.quaternion, _dummy.scale);

		// [Fix #5] Hide zero-vector arrows by scaling to 0
		if (vector.x === 0 && vector.y === 0 && vector.z === 0) {
			_dummy.scale.set(0, 0, 0);
		} else {
			_dummy.scale.set(1, 1, 1);
			// [Fix #1] Reuse _tempVec instead of allocating new Vector3 each iteration
			_tempVec.set(vector.x, vector.y, vector.z).normalize();
			_dummy.quaternion.setFromUnitVectors(_defaultUp, _tempVec);
		}

		// [Fix #9] Shared color computation
		magnetizationHSL(vector.x, vector.y, vector.z, _color);
		colors[i * 3 + 0] = _color.r;
		colors[i * 3 + 1] = _color.g;
		colors[i * 3 + 2] = _color.b;

		_dummy.updateMatrix();
		mesh.setMatrixAt(i, _dummy.matrix);
	}

	mesh.instanceMatrix.needsUpdate = true;
	instanceColor.needsUpdate = true;

	// [Fix #4] No need to update store — mesh reference hasn't changed
}

// ─── Reset camera ───────────────────────────────────────────────────
export function resetCamera() {
	const dims = [get(previewState).xChosenSize, get(previewState).yChosenSize];
	const nz = get(previewState).allLayers ? get(meshState).Nz : 1;
	const posx = dims[0] / 2;
	const posy = dims[1] / 2;
	const posz = Math.max(dims[0], dims[1], nz) * 1.5;

	const displayInstance = get(threeDPreview);
	if (displayInstance) {
		const camera = displayInstance.camera;
		const controls = displayInstance.controls;
		camera.position.set(posx, posy, posz);
		camera.up.set(0, 1, 0);
		camera.lookAt(dims[0] / 2, dims[1] / 2, nz / 2);
		controls.target.set(dims[0] / 2, dims[1] / 2, nz / 2);
		controls.update();
	}
}