<script lang="ts">
	import { meshState as mesh } from '$api/incoming/mesh';
	import Panel from '$lib/ui/Panel.svelte';
	import ReadonlyField from '$lib/ui/ReadonlyField.svelte';

	const SI_PREFIXES: { threshold: number; divisor: number; unit: string }[] = [
		{ threshold: 1, divisor: 1, unit: 'm' },
		{ threshold: 1e-3, divisor: 1e-3, unit: 'mm' },
		{ threshold: 1e-6, divisor: 1e-6, unit: 'µm' },
		{ threshold: 1e-9, divisor: 1e-9, unit: 'nm' },
		{ threshold: 0, divisor: 1e-12, unit: 'pm' }
	];

	function formatSI(meters: number): { value: string; unit: string } {
		if (meters === 0) return { value: '0', unit: 'm' };
		const abs = Math.abs(meters);
		for (const { threshold, divisor, unit } of SI_PREFIXES) {
			if (abs >= threshold) {
				const scaled = meters / divisor;
				const decimals = abs >= 1 ? 3 : Math.max(0, 4 - Math.floor(Math.log10(Math.abs(scaled)) + 1));
				return { value: scaled.toFixed(decimals), unit };
			}
		}
		const last = SI_PREFIXES[SI_PREFIXES.length - 1];
		return { value: (meters / last.divisor).toFixed(3), unit: last.unit };
	}

	const dimensions = $derived([
		{ label: 'dx', ...formatSI($mesh.dx), group: 'Cell size' },
		{ label: 'dy', ...formatSI($mesh.dy), group: 'Cell size' },
		{ label: 'dz', ...formatSI($mesh.dz), group: 'Cell size' },
		{ label: 'Nx', value: `${$mesh.Nx}`, unit: '', group: 'Grid' },
		{ label: 'Ny', value: `${$mesh.Ny}`, unit: '', group: 'Grid' },
		{ label: 'Nz', value: `${$mesh.Nz}`, unit: '', group: 'Grid' },
		{ label: 'Tx', ...formatSI($mesh.Tx), group: 'Size' },
		{ label: 'Ty', ...formatSI($mesh.Ty), group: 'Size' },
		{ label: 'Tz', ...formatSI($mesh.Tz), group: 'Size' },
		{ label: 'PBCx', value: `${$mesh.PBCx}`, unit: '', group: 'Boundaries' },
		{ label: 'PBCy', value: `${$mesh.PBCy}`, unit: '', group: 'Boundaries' },
		{ label: 'PBCz', value: `${$mesh.PBCz}`, unit: '', group: 'Boundaries' }
	]);

	const grouped = $derived(
		Array.from(
			dimensions.reduce((map, entry) => {
				const bucket = map.get(entry.group) ?? [];
				bucket.push(entry);
				map.set(entry.group, bucket);
				return map;
			}, new Map<string, typeof dimensions>())
		)
	);
</script>

<Panel
	title="Mesh"
	subtitle="Technical mesh facts exposed as true readonly values."
	panelId="mesh"
	eyebrow="Inspector"
>
	<div class="mesh-groups">
		{#each grouped as [group, entries]}
			<div class="mesh-group">
				<header>{group}</header>
				<div class="mesh-grid">
					{#each entries as entry}
						<ReadonlyField
							label={entry.label}
							value={entry.value}
							unit={entry.unit ?? ''}
							mono={true}
						/>
					{/each}
				</div>
			</div>
		{/each}
	</div>
</Panel>

<style>
	.mesh-groups {
		display: grid;
		gap: 1rem;
	}

	.mesh-group {
		display: grid;
		gap: 0.8rem;
	}

	.mesh-group header {
		font-size: 0.76rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.12em;
		color: var(--text-3);
	}

	.mesh-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.8rem;
	}

	@media (max-width: 1023px) {
		.mesh-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 639px) {
		.mesh-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
