<script lang="ts">
	import { headerState } from '$api/incoming/header';
	import { connected } from '$api/websocket';
</script>

<section class="header-bar">
	<div class="header-content">
		<!-- Connection & Status -->
		<div class="status-group">
			<div class="connection-badge" class:connected={$connected} class:disconnected={!$connected}>
				<span class="dot"></span>
				{#if $connected}
					{#if $headerState.status === 'running'}
						<span class="label">Running</span>
					{:else if $headerState.status === 'paused'}
						<span class="label">Paused</span>
					{:else}
						<span class="label">Idle</span>
					{/if}
				{:else}
					<span class="label">Disconnected</span>
				{/if}
			</div>
		</div>

		<!-- File Path -->
		<div class="file-path">
			{$headerState.path}
		</div>

		<!-- Version -->
		<div class="version">
			v{$headerState.version}
		</div>
	</div>
</section>

<style>
	.header-bar {
		position: sticky;
		top: 0;
		z-index: var(--z-sticky);
		background: var(--surface-1);
		border-bottom: 1px solid var(--border);
		padding: var(--space-sm) var(--space-lg);
		backdrop-filter: blur(12px);
	}
	.header-content {
		display: flex;
		align-items: center;
		gap: var(--space-lg);
		max-width: 100%;
	}
	.status-group {
		flex-shrink: 0;
	}
	.connection-badge {
		display: flex;
		align-items: center;
		gap: var(--space-xs);
		padding: 3px 10px;
		border-radius: 20px;
		font-size: 12px;
		font-weight: 500;
		letter-spacing: 0.02em;
		transition: all var(--duration-fast) var(--easing-default);
	}
	.connection-badge.connected {
		background: rgba(34, 197, 94, 0.12);
		color: var(--success);
		border: 1px solid rgba(34, 197, 94, 0.25);
	}
	.connection-badge.disconnected {
		background: rgba(239, 68, 68, 0.12);
		color: var(--danger);
		border: 1px solid rgba(239, 68, 68, 0.25);
	}
	.dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		flex-shrink: 0;
	}
	.connected .dot {
		background: var(--success);
		box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
	}
	.disconnected .dot {
		background: var(--danger);
		box-shadow: 0 0 6px rgba(239, 68, 68, 0.5);
	}
	.file-path {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-family: var(--font-mono);
		font-size: 13px;
		color: var(--text-2);
	}
	.version {
		flex-shrink: 0;
		font-size: 12px;
		font-family: var(--font-mono);
		color: var(--text-3);
		white-space: nowrap;
	}
</style>
