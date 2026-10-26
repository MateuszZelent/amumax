import { expect, test } from '@playwright/test';

test('renders vector data in glyph and voxel modes', async ({ page }) => {
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.message));
	await page.routeWebSocket(/\/ws/, (socket) => socket.close());
	await page.route('http://127.0.0.1:4173/api/**', (route) => route.fulfill({ json: {} }));
	await page.goto('/');
	await expect(page.locator('#container')).toBeVisible();

	const result = await page.evaluate(async () => {
		const meshPath = '/src/api/incoming/mesh.ts';
		const previewPath = '/src/api/incoming/preview.ts';
		const rendererPath = '/src/lib/preview/preview3D.ts';
		const { meshState } = await import(meshPath);
		const { previewState } = await import(previewPath);
		const renderer = await import(rendererPath);
		const values = Array.from({ length: 8 }, () => ({ x: 0, y: 1, z: 0 }));
		const positions = values.map((_, i) => ({ x: i % 2, y: (i >> 1) % 2, z: i >> 2 }));
		meshState.update((state: object) => ({ ...state, Nx: 2, Ny: 2, Nz: 2, dx: 1, dy: 1, dz: 1 }));
		previewState.update((state: object) => ({
			...state,
			quantity: 'm',
			type: '3D',
			nComp: 3,
			component: '3D',
			allLayers: true,
			vectorFieldValues: values,
			vectorFieldPositions: positions,
			appliedXChosenSize: 2,
			appliedYChosenSize: 2,
			appliedLayerStride: 1,
			refresh: true
		}));
		await new Promise(requestAnimationFrame);
		renderer.setRenderMode('glyph');
		renderer.preview3D();
		let glyph: any;
		let visible = 0;
		let unsubscribe = renderer.threeDPreview.subscribe((value: any) => (glyph = value));
		unsubscribe();
		unsubscribe = renderer.visibleRenderCount.subscribe((value: number) => (visible = value));
		unsubscribe();
		const glyphCount = glyph.mesh.count;
		const glyphVisible = visible;
		renderer.setRenderMode('voxel');
		let voxel: any;
		unsubscribe = renderer.threeDPreview.subscribe((value: any) => (voxel = value));
		unsubscribe();
		unsubscribe = renderer.visibleRenderCount.subscribe((value: number) => (visible = value));
		unsubscribe();
		return { glyphCount, glyphVisible, voxelCount: voxel.mesh.count, voxelVisible: visible };
	});

	expect(result).toEqual({ glyphCount: 8, glyphVisible: 8, voxelCount: 8, voxelVisible: 8 });
	await expect(page.locator('#container canvas')).toHaveCount(1);
	expect(errors).toEqual([]);
});
