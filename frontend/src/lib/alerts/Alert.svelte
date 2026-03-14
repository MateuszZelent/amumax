<script>
	import { alert } from './alert';
</script>

{#if $alert !== ''}
	<div class="toast-stack" role="status" aria-live="assertive">
		<article class="toast">
			<div class="toast__icon" aria-hidden="true">!</div>
			<div class="toast__body">
				<strong>Request failed</strong>
				<p>{$alert}</p>
			</div>
		</article>
	</div>
{/if}

<style>
	.toast-stack {
		position: fixed;
		top: 1rem;
		right: 1rem;
		z-index: var(--z-toast);
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.toast {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: start;
		gap: 0.85rem;
		width: min(27rem, calc(100vw - 2rem));
		padding: 0.95rem 1rem;
		border: 1px solid rgba(255, 124, 124, 0.34);
		border-radius: var(--radius-md);
		background: rgba(31, 14, 24, 0.96);
		box-shadow: var(--shadow-panel);
	}

	.toast__icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		border-radius: 50%;
		background: rgba(255, 124, 124, 0.15);
		color: var(--danger);
		font-weight: 700;
	}

	.toast__body {
		min-width: 0;
	}

	.toast__body strong {
		display: block;
		margin-bottom: 0.2rem;
	}

	.toast__body p {
		margin: 0;
		color: var(--text-2);
		overflow-wrap: anywhere;
	}
</style>
