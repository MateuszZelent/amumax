<script lang="ts">
	import type { StatusTone } from './types';

	type Props = {
		label: string;
		value: string;
		detail?: string;
		progress?: number | null;
		tone?: StatusTone;
	};

	let { label, value, detail = '', progress = null, tone = 'default' }: Props = $props();

	const normalized = $derived(progress == null ? null : Math.max(0, Math.min(progress, 100)));
</script>

<article class="ui-metric" data-tone={tone}>
	<header>
		<span>{label}</span>
		{#if detail}
			<small>{detail}</small>
		{/if}
	</header>
	<strong>{value}</strong>
	{#if normalized != null}
		<div class="ui-metric__bar" aria-hidden="true">
			<span style={`width:${normalized}%`}></span>
		</div>
	{/if}
</article>

<style>
	.ui-metric {
		display: flex;
		flex-direction: column;
		gap: 0.7rem;
		padding: 0.95rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
	}

	.ui-metric header {
		display: flex;
		justify-content: space-between;
		gap: 0.8rem;
		color: var(--text-3);
		font-size: 0.76rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.ui-metric header small {
		color: var(--text-2);
		font-size: 0.76rem;
		text-transform: none;
		letter-spacing: normal;
	}

	.ui-metric strong {
		font-size: 1.22rem;
		letter-spacing: -0.02em;
	}

	.ui-metric__bar {
		height: 0.5rem;
		border-radius: var(--radius-pill);
		background: rgba(255, 255, 255, 0.06);
		overflow: hidden;
	}

	.ui-metric__bar span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: linear-gradient(90deg, var(--info), var(--accent));
	}

	.ui-metric[data-tone='warn'] .ui-metric__bar span {
		background: linear-gradient(90deg, var(--warn), #f39b3d);
	}

	.ui-metric[data-tone='danger'] .ui-metric__bar span {
		background: linear-gradient(90deg, var(--danger), #ff5d78);
	}
</style>
