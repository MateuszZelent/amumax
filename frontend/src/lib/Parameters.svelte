<script lang="ts">
	import { parametersState as p } from '$api/incoming/parameters';
	import { postSelectedRegion } from '$api/outgoing/parameters';

	let dropdownOpen = false;
	let showZeroValues = false;
</script>

<section>
	<h2>Parameters</h2>

	<div class="toolbar">
		<!-- Region selector -->
		<div class="dropdown-wrapper">
			<button
				class="dropdown-trigger"
				on:click={() => dropdownOpen = !dropdownOpen}
			>
				<span class="dropdown-label">Region</span>
				<span class="dropdown-value">{$p.selectedRegion}</span>
				<span class="dropdown-arrow">▾</span>
			</button>
			{#if dropdownOpen}
				<div class="dropdown-menu">
					{#each $p.regions as region}
						<button
							class="dropdown-item"
							class:active={$p.selectedRegion === region}
							on:click={() => { postSelectedRegion(region); dropdownOpen = false; }}
						>
							{region}
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Toggle -->
		<label class="toggle">
			<input type="checkbox" bind:checked={showZeroValues} />
			<span class="toggle-label">Show unchanged</span>
		</label>
	</div>

	<div class="params-list">
		{#each $p.fields as field}
			{#if field.changed || showZeroValues}
				<div class="param-row">
					<span class="param-name">{field.name}</span>
					<span class="param-value">{field.value}</span>
					<span class="param-desc">{field.description}</span>
				</div>
			{/if}
		{/each}
	</div>
</section>

<style>
	.toolbar {
		display: flex;
		align-items: center;
		gap: var(--space-md);
		margin-bottom: var(--space-md);
		flex-wrap: wrap;
	}

	/* ─── Dropdown ─────────────────────────────────────────────── */
	.dropdown-wrapper {
		position: relative;
	}
	.dropdown-trigger {
		display: flex;
		align-items: center;
		padding: 6px 10px;
		gap: var(--space-sm);
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		color: var(--text-1);
		font-size: 13px;
		cursor: pointer;
		transition: border-color var(--duration-fast) var(--easing-default);
	}
	.dropdown-trigger:hover { border-color: var(--border-interactive); }
	.dropdown-label { color: var(--text-3); flex-shrink: 0; }
	.dropdown-value { font-weight: 600; font-family: var(--font-mono); }
	.dropdown-arrow { color: var(--text-3); font-size: 11px; }
	.dropdown-menu {
		position: absolute;
		top: 100%; left: 0;
		z-index: var(--z-overlay);
		margin-top: 4px;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-lg);
		max-height: 200px;
		overflow-y: auto;
		min-width: 120px;
	}
	.dropdown-item {
		display: block; width: 100%;
		padding: 5px 12px; text-align: left;
		background: transparent; border: none;
		color: var(--text-2); font-size: 13px;
		font-family: var(--font-mono); cursor: pointer;
		transition: all var(--duration-instant) var(--easing-default);
	}
	.dropdown-item:hover { background: var(--surface-3); color: var(--text-1); }
	.dropdown-item.active { color: var(--accent); font-weight: 600; }

	/* ─── Toggle ───────────────────────────────────────────────── */
	.toggle {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
		cursor: pointer;
		font-size: 13px;
	}
	.toggle input[type="checkbox"] {
		width: 16px; height: 16px;
		accent-color: var(--accent);
		cursor: pointer;
	}
	.toggle-label { color: var(--text-2); }

	/* ─── Parameters list ─────────────────────────────────────── */
	.params-list {
		display: flex;
		flex-direction: column;
		gap: 0;
	}
	.param-row {
		display: grid;
		grid-template-columns: 120px 1fr 2fr;
		gap: var(--space-sm);
		align-items: baseline;
		padding: 4px 0;
		border-bottom: 1px solid var(--border-subtle);
	}
	.param-row:last-child { border-bottom: none; }
	.param-name {
		font-family: var(--font-mono);
		font-size: 13px;
		font-weight: 500;
		color: var(--text-1);
		text-align: right;
		padding-right: var(--space-sm);
	}
	.param-value {
		font-family: var(--font-mono);
		font-size: 13px;
		color: var(--accent);
	}
	.param-desc {
		font-size: 12px;
		font-style: italic;
		color: var(--text-3);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
