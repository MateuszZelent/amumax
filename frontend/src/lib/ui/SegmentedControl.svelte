<script lang="ts">
	type Option = {
		value: string;
		label: string;
		disabled?: boolean;
	};

	type Props = {
		label?: string;
		value: string;
		options: Option[];
		onchange?: (value: string) => void;
		compact?: boolean;
	};

	let { label = '', value, options, onchange, compact = false }: Props = $props();
</script>

<div class="ui-segmented-wrapper">
	{#if label}
		<span class="ui-segmented__label">{label}</span>
	{/if}
	<div class="ui-segmented" data-compact={compact}>
		{#each options as option}
			<button
				type="button"
				class="ui-segmented__option"
				data-active={value === option.value}
				disabled={option.disabled}
				onclick={() => onchange?.(option.value)}
			>
				{option.label}
			</button>
		{/each}
	</div>
</div>

<style>
	.ui-segmented-wrapper {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
	}

	.ui-segmented__label {
		font-size: 0.76rem;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.ui-segmented {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: 1fr;
		padding: 0.24rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
	}

	.ui-segmented[data-compact='true'] {
		padding: 0.16rem;
	}

	.ui-segmented__option {
		min-height: 2.3rem;
		padding: 0 0.85rem;
		border-radius: var(--radius-pill);
		color: var(--text-2);
		font-weight: 600;
		cursor: pointer;
		transition:
			background var(--duration-fast) var(--easing-default),
			color var(--duration-fast) var(--easing-default);
	}

	.ui-segmented__option[data-active='true'] {
		background: linear-gradient(135deg, rgba(87, 200, 182, 0.95), rgba(56, 178, 162, 0.95));
		color: #09101b;
	}

	.ui-segmented__option:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}
</style>
