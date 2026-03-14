<script lang="ts">
	type Props = {
		label: string;
		checked?: boolean;
		onchange?: (next: boolean) => void;
	};

	let { label, checked = false, onchange }: Props = $props();

	function handleChange(event: Event) {
		onchange?.((event.currentTarget as HTMLInputElement).checked);
	}
</script>

<label class="ui-toggle">
	<input type="checkbox" checked={checked} onchange={handleChange} />
	<span class="ui-toggle__track" aria-hidden="true">
		<span class="ui-toggle__thumb"></span>
	</span>
	<span class="ui-toggle__label">{label}</span>
</label>

<style>
	.ui-toggle {
		display: inline-flex;
		align-items: center;
		gap: 0.75rem;
		cursor: pointer;
		color: var(--text-2);
	}

	.ui-toggle input {
		position: absolute;
		opacity: 0;
		pointer-events: none;
	}

	.ui-toggle__track {
		position: relative;
		width: 2.7rem;
		height: 1.55rem;
		border-radius: var(--radius-pill);
		background: rgba(255, 255, 255, 0.08);
		border: 1px solid var(--border-subtle);
		transition: background var(--duration-fast) var(--easing-default);
	}

	.ui-toggle__thumb {
		position: absolute;
		top: 0.16rem;
		left: 0.16rem;
		width: 1.05rem;
		height: 1.05rem;
		border-radius: 50%;
		background: var(--text-2);
		transition:
			transform var(--duration-fast) var(--easing-spring),
			background var(--duration-fast) var(--easing-default);
	}

	input:checked + .ui-toggle__track {
		background: rgba(87, 200, 182, 0.24);
	}

	input:checked + .ui-toggle__track .ui-toggle__thumb {
		transform: translateX(1.1rem);
		background: var(--accent);
	}

	.ui-toggle__label {
		font-size: 0.92rem;
	}
</style>
