<script lang="ts">
	import type { StatusTone } from './types';

	type Props = {
		label: string;
		tone?: StatusTone;
		pulse?: boolean;
		className?: string;
	};

	let { label, tone = 'default', pulse = false, className = '' }: Props = $props();
</script>

<span class={`ui-badge ${className}`.trim()} data-tone={tone} data-pulse={pulse}>
	<span class="ui-badge__dot" aria-hidden="true"></span>
	<span>{label}</span>
</span>

<style>
	.ui-badge {
		display: inline-flex;
		align-items: center;
		gap: 0.45rem;
		padding: 0.4rem 0.75rem;
		border-radius: var(--radius-pill);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
		color: var(--text-2);
		font-size: 0.82rem;
		font-weight: 600;
		white-space: nowrap;
	}

	.ui-badge__dot {
		width: 0.55rem;
		height: 0.55rem;
		border-radius: 50%;
		background: currentColor;
		box-shadow: 0 0 0 0 rgba(255, 255, 255, 0.16);
	}

	.ui-badge[data-tone='accent'],
	.ui-badge[data-tone='success'] {
		color: var(--success);
	}

	.ui-badge[data-tone='info'] {
		color: var(--info);
	}

	.ui-badge[data-tone='warn'] {
		color: var(--warn);
	}

	.ui-badge[data-tone='danger'] {
		color: var(--danger);
	}

	.ui-badge[data-pulse='true'] .ui-badge__dot {
		animation: badge-pulse 1.6s infinite;
	}

	@keyframes badge-pulse {
		0% {
			box-shadow: 0 0 0 0 currentColor;
		}

		70% {
			box-shadow: 0 0 0 8px transparent;
		}

		100% {
			box-shadow: 0 0 0 0 transparent;
		}
	}
</style>
