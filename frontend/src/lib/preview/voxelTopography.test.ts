import { describe, expect, it } from 'vitest';
import { resolveVoxelTopography } from './voxelTopography';

describe('resolveVoxelTopography', () => {
	it('keeps neutral voxels unchanged', () => {
		expect(resolveVoxelTopography(3, 0.4, 0)).toEqual({
			centerZ: 3,
			depthScale: 0.4
		});
	});

	it('extends positive topography upward while keeping the lower face fixed', () => {
		const result = resolveVoxelTopography(3, 0.4, 1.2);
		const originalBottom = 3 - 0.2;
		const newBottom = result.centerZ - result.depthScale / 2;
		const originalTop = 3 + 0.2;
		const newTop = result.centerZ + result.depthScale / 2;

		expect(newBottom).toBeCloseTo(originalBottom);
		expect(newTop).toBeCloseTo(originalTop + 1.2);
	});

	it('extends negative topography downward while keeping the upper face fixed', () => {
		const result = resolveVoxelTopography(3, 0.4, -1.2);
		const originalTop = 3 + 0.2;
		const newTop = result.centerZ + result.depthScale / 2;
		const originalBottom = 3 - 0.2;
		const newBottom = result.centerZ - result.depthScale / 2;

		expect(newTop).toBeCloseTo(originalTop);
		expect(newBottom).toBeCloseTo(originalBottom - 1.2);
	});
});
