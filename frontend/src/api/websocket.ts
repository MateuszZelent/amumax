import { decode } from '@msgpack/msgpack';

import { type Preview, previewState } from './incoming/preview';
import { type Header, headerState } from './incoming/header';
import { type Solver, solverState } from './incoming/solver';
import { type Console, consoleState } from './incoming/console';
import { type Mesh, meshState } from './incoming/mesh';
import { type Parameters, parametersState, sortFieldsByName } from './incoming/parameters';
import { type TablePlot, tablePlotState } from './incoming/table-plot';
import { get, writable } from 'svelte/store';
import { metricsState, type Metrics } from './incoming/metrics';
import { fftState, type FftData } from './incoming/fft';
import type { ConnectionState } from '$lib/ui/types';

type MainUpdate = {
	console?: Console;
	header?: Header;
	mesh?: Mesh;
	parameters?: Parameters;
	solver?: Solver;
	tablePlot?: TablePlot;
	preview?: Preview | null;
	metrics?: Metrics;
	fft?: FftData;
};

export let connected = writable(false);
export let connectionState = writable<ConnectionState>('disconnected');
let previewRenderScheduled = false;
let tableRenderScheduled = false;

async function renderPreview() {
	if (get(previewState).type === '3D') {
		const { preview3D } = await import('$lib/preview/preview3D');
		await preview3D();
		return;
	}

	const { preview2D } = await import('$lib/preview/preview2D');
	await preview2D();
}

async function renderTablePlot() {
	const { plotTable } = await import('$lib/table-plot/table-plot');
	await plotTable();
}

function schedulePreviewRender() {
	if (previewRenderScheduled) {
		return;
	}
	previewRenderScheduled = true;
	requestAnimationFrame(() => {
		previewRenderScheduled = false;
		void renderPreview();
	});
}

function scheduleTableRender() {
	if (tableRenderScheduled) {
		return;
	}
	tableRenderScheduled = true;
	requestAnimationFrame(() => {
		tableRenderScheduled = false;
		void renderTablePlot();
	});
}

function connectWS(
	wsUrl: string,
	onOpen: () => void,
	onClose: () => void,
	onMessage: (data: ArrayBuffer) => void
) {
	const retryInterval = 1000;
	let ws: WebSocket | null = null;

	function connect() {
		console.debug('Connecting to WebSocket server at', wsUrl);
		ws = new WebSocket(wsUrl);
		ws.binaryType = 'arraybuffer';

		ws.onopen = function () {
			onOpen();
		};

		ws.onmessage = function (event) {
			onMessage(event.data as ArrayBuffer);
			ws?.send('ok');
		};

		ws.onclose = function () {
			onClose();
			console.debug(
				'WebSocket closed. Attempting to reconnect in ' + retryInterval / 1000 + ' seconds...'
			);
			ws = null;
			setTimeout(connect, retryInterval);
		};

		ws.onerror = function (event) {
			console.error('WebSocket encountered error:', event);
			if (ws) {
				ws.close();
			}
		};
	}

	try {
		connect();
	} catch (err) {
		console.error(
			'WebSocket connection failed:',
			err,
			'Retrying in ' + retryInterval / 1000 + ' seconds...'
		);
		setTimeout(connect, retryInterval);
	}
}

export function initializeWebSocket() {
	connectWS(
		'./ws',
		() => {
			connected.set(true);
			connectionState.set('connected');
		},
		() => {
			connected.set(false);
			connectionState.update((state) => (state === 'connected' ? 'reconnecting' : 'disconnected'));
		},
		parseMsgpack
	);
	connectWS(
		'./ws/preview',
		() => undefined,
		() => undefined,
		parsePreviewMsgpack
	);
}

export function parseMsgpack(data: ArrayBuffer) {
	const msg = decode(new Uint8Array(data)) as MainUpdate;

	if (msg.console) {
		consoleState.set(msg.console);
	}
	if (msg.header) {
		headerState.set(msg.header);
	}
	if (msg.mesh) {
		meshState.set(msg.mesh);
	}
	if (msg.parameters) {
		parametersState.set(msg.parameters);
		sortFieldsByName();
	}
	if (msg.solver) {
		solverState.set(msg.solver);
	}
	if (msg.tablePlot) {
		tablePlotState.set(msg.tablePlot);
		scheduleTableRender();
	}
	if (msg.preview) {
		previewState.set(msg.preview);
		schedulePreviewRender();
	}
	if (msg.metrics) {
		metricsState.set(msg.metrics);
	}
	if (msg.fft) {
		fftState.set(msg.fft);
	}
}

export function parsePreviewMsgpack(data: ArrayBuffer) {
	const msg = decode(new Uint8Array(data)) as Preview;
	previewState.set(msg);
	schedulePreviewRender();
}
