<script lang="ts">
	import { consoleState } from '$api/incoming/console';
	import { postCommand } from '$api/outgoing/console';

	import Prism from 'prismjs';
	import 'prismjs/components/prism-go'; // Ensure the import path is correct
	import { onMount, tick } from 'svelte';

	let command = '';
	let codeDiv: HTMLDivElement | null = null;

	// Handle Enter key to submit commands
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			postCommand(command);
			event.preventDefault(); // Prevent the default action to avoid form submission or newline in input
		}
	}
	async function scrollDown() {
		if (codeDiv) {
			// Wait for the DOM to update (e.g. after a new line in hist is added)
			await tick();
			const isAtBottom = codeDiv.scrollTop + codeDiv.clientHeight >= codeDiv.scrollHeight - 100;
			if (isAtBottom) {
				codeDiv.scrollBy(0, codeDiv.scrollHeight);
			}
		}
	}

	$: {
		$consoleState.hist; // Trigger the reactive statement when hist changes
		scrollDown();
	}
	onMount(() => {
		scrollDown();
	});
</script>

<section>
	<h2 class="mb-4 text-2xl font-semibold">Console</h2>
	<div class="console-container">
		<div class="code" bind:this={codeDiv}>
			{@html Prism.highlight($consoleState.hist, Prism.languages['go'], 'go')}
		</div>
		<div class="input-row">
			<span class="prompt">›</span>
			<input
				placeholder="type commands here..."
				bind:value={command}
				on:keydown={handleKeydown}
			/>
		</div>
	</div>
</section>

<style>
	section {
	}
	.console-container {
		display: flex;
		flex-direction: column;
		gap: var(--space-xs);
	}
	.code {
		white-space: pre-wrap;
		overflow-y: auto;
		height: 28rem;
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		width: 100%;
		padding: var(--space-sm) var(--space-md);
		font-family: var(--font-mono);
		font-size: 13px;
		line-height: 1.6;
		color: var(--text-2);
		background-color: var(--surface-2);
	}
	.input-row {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: 0 var(--space-md);
	}
	.prompt {
		color: var(--accent);
		font-family: var(--font-mono);
		font-weight: 600;
		font-size: 14px;
		flex-shrink: 0;
	}
	.input-row input {
		width: 100%;
		background: transparent;
		border: none;
		color: var(--text-1);
		font-family: var(--font-mono);
		font-size: 13px;
		padding: var(--space-sm) 0;
	}
	.input-row input:focus {
		outline: none;
		box-shadow: none;
	}
</style>
