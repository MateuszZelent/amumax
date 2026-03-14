import { browser } from '$app/environment';
import { writable } from 'svelte/store';

import type { PanelId, PanelPreferences, CompactTab, DensityMode, ViewportMode } from './types';

const STORAGE_KEY = 'amumax:panel-prefs';

const panelIds: PanelId[] = ['preview', 'tableplot', 'solver', 'parameters', 'mesh', 'console', 'metrics'];

function defaultCollapsed() {
	return Object.fromEntries(panelIds.map((panelId) => [panelId, false])) as Record<PanelId, boolean>;
}

const defaults: PanelPreferences = {
	collapsedByPanel: defaultCollapsed(),
	activeCompactTab: 'visualization',
	diagnosticsHeight: 352,
	preferredPreviewMode: 'inline',
	densityMode: 'compact'
};

function sanitize(input: Partial<PanelPreferences> | null | undefined): PanelPreferences {
	const collapsedByPanel = { ...defaultCollapsed(), ...(input?.collapsedByPanel ?? {}) };
	const activeCompactTab = (input?.activeCompactTab ?? defaults.activeCompactTab) as CompactTab;
	const preferredPreviewMode = (input?.preferredPreviewMode ??
		defaults.preferredPreviewMode) as ViewportMode;
	const densityMode = (input?.densityMode ?? defaults.densityMode) as DensityMode;

	return {
		collapsedByPanel,
		activeCompactTab,
		diagnosticsHeight: Math.max(280, Math.min(input?.diagnosticsHeight ?? defaults.diagnosticsHeight, 640)),
		preferredPreviewMode,
		densityMode
	};
}

function loadPreferences(): PanelPreferences {
	if (!browser) {
		return defaults;
	}

	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) {
			return defaults;
		}
		return sanitize(JSON.parse(raw));
	} catch {
		return defaults;
	}
}

export const panelPreferences = writable<PanelPreferences>(loadPreferences());

if (browser) {
	panelPreferences.subscribe((value) => {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
		document.body.dataset.density = value.densityMode;
	});
}

export function setCollapsed(panelId: PanelId, collapsed: boolean) {
	panelPreferences.update((prefs) => ({
		...prefs,
		collapsedByPanel: {
			...prefs.collapsedByPanel,
			[panelId]: collapsed
		}
	}));
}

export function toggleCollapsed(panelId: PanelId) {
	panelPreferences.update((prefs) => ({
		...prefs,
		collapsedByPanel: {
			...prefs.collapsedByPanel,
			[panelId]: !prefs.collapsedByPanel[panelId]
		}
	}));
}

export function setActiveCompactTab(activeCompactTab: CompactTab) {
	panelPreferences.update((prefs) => ({ ...prefs, activeCompactTab }));
}

export function setPreferredPreviewMode(preferredPreviewMode: ViewportMode) {
	panelPreferences.update((prefs) => ({ ...prefs, preferredPreviewMode }));
}

export function setDensityMode(densityMode: DensityMode) {
	panelPreferences.update((prefs) => ({ ...prefs, densityMode }));
}

export function setDiagnosticsHeight(diagnosticsHeight: number) {
	panelPreferences.update((prefs) => ({ ...prefs, diagnosticsHeight }));
}
