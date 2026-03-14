<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { StatusTone } from './types';

	type Props = {
		title: string;
		description: string;
		tone?: StatusTone;
		compact?: boolean;
		children?: Snippet;
	};

	let { title, description, tone = 'default', compact = false, children }: Props = $props();
</script>

<div class="ui-empty" data-tone={tone} data-compact={compact}>
	<div class="ui-empty__eyebrow">{tone === 'danger' ? 'Attention' : 'State'}</div>
	<h3>{title}</h3>
	<p>{description}</p>
	<div class="ui-empty__actions">
		{#if children}
			{@render children()}
		{/if}
	</div>
</div>

<style>
	.ui-empty {
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		padding: 1rem;
		border: 1px dashed var(--border);
		border-radius: var(--radius-md);
		background: rgba(255, 255, 255, 0.025);
		color: var(--text-2);
		text-align: left;
	}

	.ui-empty[data-compact='true'] {
		padding: 0.85rem;
	}

	.ui-empty__eyebrow {
		font-size: 0.7rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.14em;
		color: var(--text-3);
	}

	.ui-empty h3 {
		margin: 0;
		font-size: 0.98rem;
		color: var(--text-1);
	}

	.ui-empty p {
		margin: 0;
	}

	.ui-empty[data-tone='danger'] {
		border-color: rgba(255, 124, 124, 0.32);
		background: rgba(255, 124, 124, 0.06);
	}

	.ui-empty[data-tone='warn'] {
		border-color: rgba(242, 180, 90, 0.32);
		background: rgba(242, 180, 90, 0.07);
	}

	.ui-empty[data-tone='info'] {
		border-color: rgba(107, 167, 255, 0.28);
		background: rgba(107, 167, 255, 0.06);
	}

	.ui-empty__actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}
</style>
