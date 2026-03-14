<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { StatusTone } from './types';

	type Props = {
		title: string;
		subtitle?: string;
		eyebrow?: string;
		tone?: StatusTone;
		panelId?: string;
		collapsible?: boolean;
		collapsed?: boolean;
		className?: string;
		onToggleCollapse?: () => void;
		children?: Snippet;
		actions?: Snippet;
	};

	let {
		title,
		subtitle = '',
		eyebrow = '',
		tone = 'default',
		panelId = '',
		collapsible = false,
		collapsed = false,
		className = '',
		onToggleCollapse,
		children,
		actions
	}: Props = $props();
</script>

<section class={`ui-panel ${className}`.trim()} data-tone={tone} data-panel={panelId} data-collapsed={collapsed}>
	<header class="ui-panel__header">
		<div class="ui-panel__heading">
			{#if eyebrow}
				<p class="ui-panel__eyebrow">{eyebrow}</p>
			{/if}
			<div class="ui-panel__titles">
				<h2>{title}</h2>
				{#if subtitle}
					<p>{subtitle}</p>
				{/if}
			</div>
		</div>

		<div class="ui-panel__actions">
			{#if actions}
				{@render actions()}
			{/if}
			{#if collapsible}
				<button
					class="ui-panel__collapse"
					type="button"
					onclick={onToggleCollapse}
					aria-label={collapsed ? `Expand ${title}` : `Collapse ${title}`}
					title={collapsed ? `Expand ${title}` : `Collapse ${title}`}
				>
					{collapsed ? '+' : '–'}
				</button>
			{/if}
		</div>
	</header>

	{#if !collapsed}
		<div class="ui-panel__body">
			{#if children}
				{@render children()}
			{/if}
		</div>
	{/if}
</section>

<style>
	.ui-panel {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 1rem;
		padding: 1rem;
		border: 1px solid var(--border-subtle);
		border-radius: var(--radius-lg);
		background:
			linear-gradient(180deg, rgba(21, 31, 51, 0.98), rgba(13, 22, 37, 0.98)),
			var(--surface-1);
		box-shadow: var(--shadow-soft);
		min-width: 0;
	}

	.ui-panel::before {
		content: '';
		position: absolute;
		inset: 0;
		border-radius: inherit;
		padding: 1px;
		background: linear-gradient(160deg, rgba(107, 167, 255, 0.2), transparent 45%, rgba(87, 200, 182, 0.16));
		mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
		mask-composite: exclude;
		opacity: 0.8;
		pointer-events: none;
	}

	.ui-panel[data-tone='accent']::before,
	.ui-panel[data-tone='success']::before {
		background: linear-gradient(160deg, rgba(87, 200, 182, 0.34), transparent 56%);
	}

	.ui-panel[data-tone='info']::before {
		background: linear-gradient(160deg, rgba(107, 167, 255, 0.36), transparent 56%);
	}

	.ui-panel[data-tone='warn']::before {
		background: linear-gradient(160deg, rgba(242, 180, 90, 0.34), transparent 56%);
	}

	.ui-panel[data-tone='danger']::before {
		background: linear-gradient(160deg, rgba(255, 124, 124, 0.34), transparent 56%);
	}

	.ui-panel__header {
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
		justify-content: space-between;
		min-width: 0;
	}

	.ui-panel__heading {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		min-width: 0;
	}

	.ui-panel__eyebrow {
		margin: 0;
		font-size: 0.72rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.12em;
		color: var(--text-3);
	}

	.ui-panel__titles h2 {
		margin: 0;
		font-size: 1.02rem;
		font-weight: 700;
		letter-spacing: -0.02em;
	}

	.ui-panel__titles p {
		margin: 0.18rem 0 0;
		font-size: 0.88rem;
		color: var(--text-2);
	}

	.ui-panel__actions {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		flex-wrap: wrap;
		justify-content: flex-end;
	}

	.ui-panel__collapse {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 2rem;
		height: 2rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
		color: var(--text-2);
		cursor: pointer;
		transition:
			border-color var(--duration-fast) var(--easing-default),
			color var(--duration-fast) var(--easing-default),
			background var(--duration-fast) var(--easing-default);
	}

	.ui-panel__collapse:hover {
		border-color: var(--border-interactive);
		color: var(--text-1);
		background: rgba(107, 167, 255, 0.08);
	}

	.ui-panel__body {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		min-width: 0;
	}
</style>
