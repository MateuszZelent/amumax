<script lang="ts">
	import { meshState as mesh } from '$api/incoming/mesh';
	import Panel from '$lib/ui/Panel.svelte';
	import ReadonlyField from '$lib/ui/ReadonlyField.svelte';

	const dimensions = $derived([
		{ label: 'dx', value: $mesh.dx.toPrecision(8), unit: 'm', group: 'Cell size' },
		{ label: 'dy', value: $mesh.dy.toPrecision(8), unit: 'm', group: 'Cell size' },
		{ label: 'dz', value: $mesh.dz.toPrecision(8), unit: 'm', group: 'Cell size' },
		{ label: 'Nx', value: `${$mesh.Nx}`, group: 'Grid' },
		{ label: 'Ny', value: `${$mesh.Ny}`, group: 'Grid' },
		{ label: 'Nz', value: `${$mesh.Nz}`, group: 'Grid' },
		{ label: 'Tx', value: $mesh.Tx.toExponential(6), unit: 'm', group: 'Size' },
		{ label: 'Ty', value: $mesh.Ty.toExponential(6), unit: 'm', group: 'Size' },
		{ label: 'Tz', value: $mesh.Tz.toExponential(6), unit: 'm', group: 'Size' },
		{ label: 'PBCx', value: `${$mesh.PBCx}`, group: 'Boundaries' },
		{ label: 'PBCy', value: `${$mesh.PBCy}`, group: 'Boundaries' },
		{ label: 'PBCz', value: `${$mesh.PBCz}`, group: 'Boundaries' }
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
