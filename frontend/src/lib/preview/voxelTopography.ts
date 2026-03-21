export interface VoxelTopographyTransform {
	centerZ: number;
	depthScale: number;
}

const TOPO_EPSILON = 1e-6;

export function resolveVoxelTopography(
	baseZ: number,
	baseDepth: number,
	signedDisplacement: number
): VoxelTopographyTransform {
	if (!Number.isFinite(signedDisplacement) || Math.abs(signedDisplacement) < TOPO_EPSILON) {
		return {
			centerZ: baseZ,
			depthScale: baseDepth
		};
	}

	return {
		centerZ: baseZ + signedDisplacement / 2,
		depthScale: baseDepth + Math.abs(signedDisplacement)
	};
}
