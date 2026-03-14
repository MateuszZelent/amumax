<script lang="ts">
	type Props = {
		label: string;
		value: string;
		unit?: string;
		hint?: string;
		mono?: boolean;
		copyValue?: string;
	};

	let {
		label,
		value,
		unit = '',
		hint = '',
		mono = false,
		copyValue = ''
	}: Props = $props();

	let copied = $state(false);

	async function copyField() {
		if (!copyValue || typeof navigator === 'undefined' || !navigator.clipboard) {
			return;
		}
		await navigator.clipboard.writeText(copyValue);
		copied = true;
		window.setTimeout(() => {
			copied = false;
		}, 1100);
	}
</script>

<div class="ui-readonly">
	<div class="ui-readonly__meta">
		<span>{label}</span>
		{#if hint}
			<span class="ui-readonly__hint">{hint}</span>
		{/if}
	</div>
	<div class="ui-readonly__value" data-mono={mono}>
		<strong>{value}</strong>
		{#if unit}
			<span>{unit}</span>
		{/if}
		{#if copyValue}
			<button type="button" class="ui-readonly__copy" onclick={copyField}>
				{copied ? 'Copied' : 'Copy'}
			</button>
		{/if}
	</div>
</div>

<style>
	.ui-readonly {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		padding: 0.85rem 0.95rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
		min-width: 0;
	}

	.ui-readonly__meta {
		display: flex;
		justify-content: space-between;
		gap: 0.75rem;
		font-size: 0.76rem;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--text-3);
	}

	.ui-readonly__hint {
		text-transform: none;
		letter-spacing: normal;
		font-size: 0.76rem;
	}

	.ui-readonly__value {
		display: flex;
		align-items: baseline;
		gap: 0.5rem;
		flex-wrap: wrap;
		min-width: 0;
	}

	.ui-readonly__value strong {
		font-size: 0.98rem;
		font-weight: 600;
		color: var(--text-1);
		min-width: 0;
		overflow-wrap: anywhere;
	}

	.ui-readonly__value span {
		color: var(--text-2);
		font-size: 0.88rem;
	}

	.ui-readonly__value[data-mono='true'] strong {
		font-family: 'IBM Plex Mono', monospace;
	}

	.ui-readonly__copy {
		margin-left: auto;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border);
		padding: 0.2rem 0.65rem;
		color: var(--text-2);
		cursor: pointer;
		transition:
			border-color var(--duration-fast) var(--easing-default),
			color var(--duration-fast) var(--easing-default);
	}

	.ui-readonly__copy:hover {
		border-color: var(--border-interactive);
		color: var(--text-1);
	}
</style>
