<script lang="ts">
	export type SelectOption = {
		value: string;
		label: string;
		group?: string;
		disabled?: boolean;
	};

	type Props = {
		label?: string;
		value: string | number;
		options: SelectOption[];
		hint?: string;
		disabled?: boolean;
		onchange?: (value: string) => void;
	};

	let { label = '', value, options, hint = '', disabled = false, onchange }: Props = $props();

	function handleChange(event: Event) {
		onchange?.((event.currentTarget as HTMLSelectElement).value);
	}
</script>

<label class="ui-select">
	{#if label}
		<span class="ui-select__label">
			<span>{label}</span>
			{#if hint}
				<span class="ui-select__hint">{hint}</span>
			{/if}
		</span>
	{/if}
	<span class="ui-select__control">
		<select {disabled} value={String(value)} onchange={handleChange}>
			{#each options as option}
				<option value={option.value} disabled={option.disabled}>
					{option.group ? `${option.group} / ${option.label}` : option.label}
				</option>
			{/each}
		</select>
		<span class="ui-select__chevron" aria-hidden="true">⌄</span>
	</span>
</label>

<style>
	.ui-select {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		min-width: 0;
	}

	.ui-select__label {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		font-size: 0.76rem;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.ui-select__hint {
		text-transform: none;
		letter-spacing: normal;
		font-size: 0.76rem;
	}

	.ui-select__control {
		position: relative;
		display: flex;
		align-items: center;
		min-height: 2.85rem;
		padding: 0 0.9rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
	}

	.ui-select__control:focus-within {
		border-color: var(--border-interactive);
		box-shadow: var(--focus-ring);
	}

	.ui-select select {
		width: 100%;
		min-width: 0;
		border: 0;
		background: transparent;
		color: var(--text-1);
		color-scheme: dark;
		outline: none;
		padding-right: 1.2rem;
		appearance: none;
	}

	.ui-select select option {
		background: var(--surface-2, #141f33);
		color: var(--text-1, #edf3fb);
	}

	.ui-select__chevron {
		position: absolute;
		right: 0.95rem;
		color: var(--text-3);
		pointer-events: none;
	}
</style>
