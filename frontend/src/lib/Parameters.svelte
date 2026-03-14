<script lang="ts">
	import { parametersState as parameters } from '$api/incoming/parameters';
	import { postSelectedRegion } from '$api/outgoing/parameters';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import SelectField from '$lib/ui/SelectField.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import Toggle from '$lib/ui/Toggle.svelte';

	let search = $state('');
	let showAll = $state(false);

	function inferGroup(name: string) {
		if (/^(Aex|Msat|alpha|Ku|Kc|Dbulk|Dind|Lambda|Pol|Temp|B1|B2)/.test(name)) {
			return 'Material';
		}
		if (/^(B_|Edens_|ext_|J|I_oersted|J_oersted|torque|LLtorque)/.test(name)) {
			return 'Fields & energy';
		}
		if (/^(geom|region|frozenspins|NoDemagSpins|MFM)/.test(name)) {
			return 'Regions & geometry';
		}
		return 'Other';
	}

	const regionOptions = $derived($parameters.regions.map((region) => ({ value: String(region), label: `Region ${region}` })));

	const visibleFields = $derived(
		$parameters.fields.filter((field) => {
			if (!showAll && !field.changed) {
				return false;
			}

			if (!search.trim()) {
				return true;
			}

			const term = search.trim().toLowerCase();
			return (
				field.name.toLowerCase().includes(term) ||
				field.description.toLowerCase().includes(term) ||
				field.value.toLowerCase().includes(term)
			);
		})
	);

	const groupedFields = $derived(
		Array.from(
			visibleFields.reduce((map, field) => {
				const group = inferGroup(field.name);
				const bucket = map.get(group) ?? [];
				bucket.push(field);
				map.set(group, bucket);
				return map;
			}, new Map<string, typeof visibleFields>())
		)
	);
</script>

<Panel
	title="Parameters"
	subtitle="Searchable inspector with changed/default filtering and grouped readonly values."
	panelId="parameters"
	eyebrow="Inspector"
>
	<div class="parameter-toolbar">
		<SelectField
			label="Region"
			value={$parameters.selectedRegion}
			options={regionOptions}
			onchange={(value) => postSelectedRegion(Number(value))}
		/>
		<TextField
			label="Search"
			placeholder="Filter by name, description or value"
			value={search}
			oninput={(event) => (search = (event.currentTarget as HTMLInputElement).value)}
		/>
		<div class="parameter-toggle">
			<Toggle label="Show unchanged values" checked={showAll} onchange={(next) => (showAll = next)} />
		</div>
	</div>

	{#if !groupedFields.length}
		<EmptyState
			title="No parameters match the current filters"
			description="Clear the search term or show unchanged values to broaden the inspector."
			tone="info"
		/>
	{:else}
		<div class="parameter-groups">
			{#each groupedFields as [group, fields]}
				<section class="parameter-group">
					<header>
						<h3>{group}</h3>
						<p>{fields.length} item{fields.length === 1 ? '' : 's'}</p>
					</header>
					<div class="parameter-list">
						{#each fields as field}
							<article class="parameter-card" data-changed={field.changed}>
								<div class="parameter-card__topline">
									<strong>{field.name}</strong>
									<span>{field.changed ? 'Changed' : 'Default'}</span>
								</div>
								<div class="parameter-card__value">{field.value}</div>
								<p>{field.description}</p>
							</article>
						{/each}
					</div>
				</section>
			{/each}
		</div>
	{/if}
</Panel>

<style>
	.parameter-toolbar {
		display: grid;
		grid-template-columns: minmax(0, 14rem) minmax(0, 1fr) auto;
		gap: 0.9rem;
		align-items: end;
	}

	.parameter-toggle {
		display: flex;
		justify-content: flex-end;
		padding-bottom: 0.15rem;
	}

	.parameter-groups {
		display: grid;
		gap: 1rem;
	}

	.parameter-group {
		display: grid;
		gap: 0.8rem;
	}

	.parameter-group header {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		align-items: baseline;
	}

	.parameter-group h3 {
		margin: 0;
		font-size: 0.96rem;
	}

	.parameter-group p {
		margin: 0;
		color: var(--text-2);
		font-size: 0.85rem;
	}

	.parameter-list {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.8rem;
	}

	.parameter-card {
		display: flex;
		flex-direction: column;
		gap: 0.55rem;
		padding: 0.9rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
		min-width: 0;
	}

	.parameter-card[data-changed='true'] {
		border-color: rgba(87, 200, 182, 0.3);
		background: rgba(87, 200, 182, 0.05);
	}

	.parameter-card__topline {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		align-items: baseline;
	}

	.parameter-card__topline strong {
		font-size: 0.95rem;
	}

	.parameter-card__topline span {
		font-size: 0.76rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: var(--text-3);
	}

	.parameter-card__value {
		font-family: 'IBM Plex Mono', monospace;
		font-size: 0.92rem;
		color: var(--text-1);
		overflow-wrap: anywhere;
	}

	@media (max-width: 1023px) {
		.parameter-toolbar {
			grid-template-columns: 1fr;
		}

		.parameter-toggle {
			justify-content: flex-start;
		}
	}

	@media (max-width: 767px) {
		.parameter-list {
			grid-template-columns: 1fr;
		}
	}
</style>
