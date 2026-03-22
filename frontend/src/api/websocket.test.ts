import { encode } from '@msgpack/msgpack';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';

vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
	callback(0);
	return 1;
});

vi.mock('$lib/preview/preview3D', () => ({
	preview3D: vi.fn(),
	threeDPreview: { subscribe: () => () => {} }
}));

vi.mock('$lib/preview/preview2D', () => ({
	preview2D: vi.fn()
}));

vi.mock('$lib/table-plot/table-plot', () => ({
	plotTable: vi.fn()
}));

import { consoleState } from './incoming/console';
import { headerState } from './incoming/header';
import { meshState } from './incoming/mesh';
import { metricsState } from './incoming/metrics';
import { parametersState } from './incoming/parameters';
import { previewState } from './incoming/preview';
import { solverState } from './incoming/solver';
import { tablePlotState } from './incoming/table-plot';
import { parseMsgpack, parsePreviewMsgpack } from './websocket';

function toArrayBuffer(payload: unknown) {
	const bytes = encode(payload);
	return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
}

function resetStores() {
	consoleState.set({ hist: '' });
	headerState.set({ path: '', status: '', version: '' });
	meshState.set({
		dx: 0,
		dy: 0,
		dz: 0,
		Nx: 0,
		Ny: 0,
		Nz: 0,
		Tx: 0,
		Ty: 0,
		Tz: 0,
		PBCx: 0,
		PBCy: 0,
		PBCz: 0
	});
	parametersState.set({ regions: [], fields: [], selectedRegion: 0 });
	solverState.set({
		type: '',
		steps: 0,
		time: 0,
		dt: 0,
		errPerStep: 0,
		maxTorque: 0,
		fixdt: 0,
		mindt: 0,
		maxdt: 0,
		maxerr: 0
	});
	tablePlotState.set({
		autoSaveInterval: 0,
		columns: [],
		xColumn: 't',
		yColumn: 'mx',
		xColumnUnit: 's',
		yColumnUnit: '',
		data: [],
		xmin: 0,
		xmax: 0,
		ymin: 0,
		ymax: 0,
		maxPoints: 0,
		step: 0,
		corePos: null
	});
	previewState.set({
		quantity: '',
		unit: '',
		component: '',
		layer: 0,
		allLayers: false,
		type: '',
		vectorFieldValues: [],
		vectorFieldPositions: [],
		scalarField: [],
		min: 0,
		max: 0,
		refresh: false,
		nComp: 0,
		maxPoints: 0,
		dataPointsCount: 0,
		xPossibleSizes: [],
		yPossibleSizes: [],
		xChosenSize: 0,
		yChosenSize: 0,
		appliedXChosenSize: 0,
		appliedYChosenSize: 0,
		appliedLayerStride: 1,
		autoScaleEnabled: true,
		autoDownscaled: false,
		autoDownscaleMessage: ''
	});
	metricsState.set({
		pid: 0,
		error: '',
		cpuPercent: 0,
		cpuPercentTotal: 0,
		ramPercent: 0,
		ramPercentTotal: 0,
		gpuName: '',
		gpuUtilizationPercent: 0,
		gpuUUID: '',
		gpuTemperature: 0,
		gpuPowerDraw: 0,
		gpuPowerLimit: 0,
		gpuVramUsed: 0,
		gpuVramTotal: 0
	});
}

describe('websocket parsing', () => {
	afterEach(resetStores);

	it('updates all main stores from a msgpack payload', () => {
		parseMsgpack(
			toArrayBuffer({
				console: { hist: 'Run()\n' },
				header: { path: '/tmp/job.mx3', status: 'running', version: '1.2.3' },
				mesh: {
					dx: 1,
					dy: 2,
					dz: 3,
					Nx: 10,
					Ny: 11,
					Nz: 12,
					Tx: 4,
					Ty: 5,
					Tz: 6,
					PBCx: 0,
					PBCy: 1,
					PBCz: 2
				},
				parameters: {
					regions: [0, 1],
					selectedRegion: 1,
					fields: [
						{ name: 'Msat', value: '1e6', description: 'Saturation', changed: true },
						{ name: 'Aex', value: '2e-11', description: 'Exchange', changed: false }
					]
				},
				solver: {
					type: 'rk45',
					steps: 12,
					time: 1e-9,
					dt: 1e-12,
					errPerStep: 0.1,
					maxTorque: 0.2,
					fixdt: 0,
					mindt: 0,
					maxdt: 0,
					maxerr: 1e-5
				},
				tablePlot: {
					autoSaveInterval: 1,
					columns: ['t', 'mx'],
					xColumn: 't',
					yColumn: 'mx',
					xColumnUnit: 's',
					yColumnUnit: '',
					data: [],
					xmin: 0,
					xmax: 1,
					ymin: -1,
					ymax: 1,
					maxPoints: 100,
					step: 2
				},
				metrics: {
					pid: 42,
					error: '',
					cpuPercent: 20,
					cpuPercentTotal: 60,
					ramPercent: 10,
					ramPercentTotal: 50,
					gpuName: 'RTX',
					gpuUtilizationPercent: 70,
					gpuUUID: 'gpu-1',
					gpuTemperature: 60,
					gpuPowerDraw: 80,
					gpuPowerLimit: 100,
					gpuVramUsed: 1024,
					gpuVramTotal: 8192
				}
			})
		);

		expect(get(consoleState).hist).toContain('Run()');
		expect(get(headerState)).toMatchObject({
			path: '/tmp/job.mx3',
			status: 'running',
			version: '1.2.3'
		});
		expect(get(meshState)).toMatchObject({ Nx: 10, Ny: 11, Nz: 12 });
		expect(get(parametersState).selectedRegion).toBe(1);
		expect(get(parametersState).fields[0].name).toBe('Aex');
		expect(get(solverState).type).toBe('rk45');
		expect(get(tablePlotState).xColumn).toBe('t');
		expect(get(metricsState).pid).toBe(42);
	});

	it('updates preview state from the dedicated preview channel', () => {
		parsePreviewMsgpack(
			toArrayBuffer({
				quantity: 'm',
				unit: 'A/m',
				component: 'x',
				layer: 2,
				allLayers: false,
				type: '2D',
				vectorFieldValues: [],
				vectorFieldPositions: [],
				scalarField: [[1, 2, 3]],
				min: -1,
				max: 1,
				refresh: false,
				nComp: 3,
				maxPoints: 64,
				dataPointsCount: 64,
				xPossibleSizes: [16, 32],
				yPossibleSizes: [16, 32],
				xChosenSize: 32,
				yChosenSize: 32,
				appliedXChosenSize: 16,
				appliedYChosenSize: 16,
				appliedLayerStride: 1,
				autoScaleEnabled: true,
				autoDownscaled: true,
				autoDownscaleMessage: 'Preview auto-scaled from 32x32 to 16x16 to stay within 128 points'
			})
		);

		expect(get(previewState)).toMatchObject({
			quantity: 'm',
			component: 'x',
			type: '2D',
			layer: 2,
			xChosenSize: 32,
			yChosenSize: 32
		});
	});
});
