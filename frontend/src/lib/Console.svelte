<script lang="ts">
	import { consoleState } from '$api/incoming/console';
	import { postCommand } from '$api/outgoing/console';
	import { connectionState } from '$api/websocket';
	import Button from '$lib/ui/Button.svelte';
	import EmptyState from '$lib/ui/EmptyState.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import Prism from 'prismjs';
	import 'prismjs/components/prism-go';
	import { tick } from 'svelte';

	let command = $state('');
	let commandHistory = $state<string[]>([]);
	let historyIndex = $state(-1);
	let codeDiv: HTMLDivElement | null = null;

	const highlightedConsole = $derived(Prism.highlight($consoleState.hist, Prism.languages['go'], 'go'));

	async function scrollDown() {
		if (!codeDiv) {
			return;
		}
		await tick();
		const isNearBottom = codeDiv.scrollTop + codeDiv.clientHeight >= codeDiv.scrollHeight - 96;
		if (isNearBottom) {
			codeDiv.scrollTop = codeDiv.scrollHeight;
		}
	}

	function submitCommand() {
		const next = command.trim();
		if (!next || $connectionState === 'disconnected') {
			return;
		}
		postCommand(next);
		commandHistory = [next, ...commandHistory.filter((entry) => entry !== next)].slice(0, 30);
		historyIndex = -1;
		command = '';
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			submitCommand();
			return;
		}

		if (event.key === 'ArrowUp') {
			event.preventDefault();
			if (!commandHistory.length) {
				return;
			}
			historyIndex = Math.min(historyIndex + 1, commandHistory.length - 1);
			command = commandHistory[historyIndex] ?? command;
		}

		if (event.key === 'ArrowDown') {
			event.preventDefault();
			if (!commandHistory.length) {
				return;
			}
			historyIndex = Math.max(historyIndex - 1, -1);
			command = historyIndex === -1 ? '' : (commandHistory[historyIndex] ?? '');
		}
	}

	$effect(() => {
		$consoleState.hist;
		scrollDown();
	});
</script>

<Panel title="Console" subtitle="Terminal-like command channel with history and sticky input." panelId="console" eyebrow="Diagnostics">
	<div class="console-shell">
		<div class="console-shell__output" bind:this={codeDiv}>
			{#if !$consoleState.hist}
				<EmptyState
					title={$connectionState === 'disconnected' ? 'Console offline' : 'Console is ready'}
					description={$connectionState === 'disconnected'
						? 'Reconnect to the backend to restore the command channel.'
						: 'Run commands to inspect or steer the simulation engine.'}
					tone={$connectionState === 'disconnected' ? 'warn' : 'info'}
					compact={true}
				/>
			{:else}
				<div class="console-shell__code">{@html highlightedConsole}</div>
			{/if}
		</div>

		<div class="console-shell__input">
			<input
				placeholder={$connectionState === 'connected'
					? 'Run a command. Use ↑ and ↓ for history.'
					: 'Console disabled while disconnected.'}
				value={command}
				oninput={(event) => (command = (event.currentTarget as HTMLInputElement).value)}
				onkeydown={handleKeydown}
				disabled={$connectionState === 'disconnected'}
			/>
			<Button variant="solid" tone="accent" onclick={submitCommand} disabled={$connectionState === 'disconnected'}>
				Send
			</Button>
		</div>
	</div>
</Panel>

<style>
	.console-shell {
		display: flex;
		flex-direction: column;
		gap: 0.9rem;
		min-height: 0;
	}

	.console-shell__output {
		min-height: var(--terminal-height);
		max-height: var(--terminal-height);
		overflow: auto;
		padding: 1rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background:
			linear-gradient(180deg, rgba(8, 12, 22, 0.96), rgba(7, 10, 18, 0.98)),
			#060911;
		box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
	}

	.console-shell__code {
		white-space: pre-wrap;
		font-family: 'IBM Plex Mono', monospace;
		font-size: 0.85rem;
		line-height: 1.6;
		color: var(--text-1);
	}

	.console-shell__input {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 0.75rem;
		position: sticky;
		bottom: 0;
	}

	.console-shell__input input {
		min-height: 2.8rem;
		padding: 0 0.95rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
		color: var(--text-1);
		outline: none;
	}

	.console-shell__input input:focus {
		border-color: var(--border-interactive);
		box-shadow: var(--focus-ring);
	}

	.console-shell__input input::placeholder {
		color: var(--text-3);
	}

	:global(.token.comment),
	:global(.token.prolog),
	:global(.token.doctype),
	:global(.token.cdata) {
		color: #7b8fb6;
	}

	:global(.token.keyword) {
		color: #7cb5ff;
	}

	:global(.token.string),
	:global(.token.attr-value) {
		color: #7ad5a4;
	}

	:global(.token.number),
	:global(.token.boolean) {
		color: #ffbf73;
	}
</style>
