export type PanelId =
	| 'preview'
	| 'tableplot'
	| 'solver'
	| 'parameters'
	| 'mesh'
	| 'console'
	| 'metrics';

export type WorkspaceMode = 'wide' | 'compact' | 'mobile';
export type CompactTab = 'visualization' | 'controls' | 'inspectors' | 'diagnostics';
export type ViewportMode = 'inline' | 'popout' | 'fullscreen';
export type DensityMode = 'compact' | 'cozy';
export type StatusTone = 'default' | 'accent' | 'info' | 'warn' | 'danger' | 'success';
export type ConnectionState = 'connected' | 'reconnecting' | 'disconnected';

export interface PanelPreferences {
	collapsedByPanel: Record<PanelId, boolean>;
	activeCompactTab: CompactTab;
	diagnosticsHeight: number;
	preferredPreviewMode: ViewportMode;
	densityMode: DensityMode;
}

export interface ThemeTokens {
	bg: string;
	surface1: string;
	surface2: string;
	surface3: string;
	border: string;
	text1: string;
	text2: string;
	accent: string;
	info: string;
	warn: string;
	danger: string;
	success: string;
}

export interface VisualizationTheme {
	background: string;
	panelBackground: string;
	border: string;
	text: string;
	mutedText: string;
	accent: string;
	info: string;
}
