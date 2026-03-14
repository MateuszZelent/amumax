<script lang="ts">
	type Props = {
		label: string;
		values?: number[];
		value: number;
		onChangeFunction: (value: number) => void;
		isDisabled?: boolean;
	};

	let { label, values = [], value, onChangeFunction, isDisabled = false }: Props = $props();

	const sliderMax = $derived(Math.max(values.length - 1, 0));

	let index = $state(0);

	$effect(() => {
		if (!values.length) {
			index = 0;
			return;
		}
		const nextIndex = values.indexOf(value);
		index = nextIndex === -1 ? 0 : nextIndex;
	});

	const currentValue = $derived(values[index] ?? values[0] ?? 0);
	const fillPercent = $derived(sliderMax === 0 ? 100 : (index / sliderMax) * 100);
</script>

<label class="slider-field" data-disabled={isDisabled}>
	<span class="slider-field__label">{label}</span>
	<div class="slider-field__control">
		<div class="slider-field__track" aria-hidden="true">
			<div class="slider-field__fill" style={`width:${fillPercent}%`}></div>
		</div>
		<input
			type="range"
			min="0"
			max={sliderMax}
			step="1"
			value={index}
			disabled={isDisabled || values.length === 0}
			oninput={(event) => {
				index = Number((event.currentTarget as HTMLInputElement).value);
			}}
			onchange={() => onChangeFunction(currentValue)}
		/>
		<div class="slider-field__meta">
			<span>{values[0] ?? 0}</span>
			<strong>{currentValue}</strong>
			<span>{values[values.length - 1] ?? 0}</span>
		</div>
	</div>
</label>

<style>
	.slider-field {
		display: flex;
		flex-direction: column;
		gap: 0.45rem;
		min-width: 0;
	}

	.slider-field__label {
		font-size: 0.76rem;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.slider-field__control {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 0.65rem;
		padding: 0.9rem;
		border-radius: var(--radius-md);
		border: 1px solid var(--border-subtle);
		background: rgba(255, 255, 255, 0.03);
	}

	.slider-field__track {
		position: absolute;
		inset: 0;
		border-radius: inherit;
		overflow: hidden;
		opacity: 0.85;
	}

	.slider-field__fill {
		height: 100%;
		background: linear-gradient(90deg, rgba(107, 167, 255, 0.16), rgba(87, 200, 182, 0.2));
		transition: width var(--duration-fast) var(--easing-default);
	}

	.slider-field input {
		position: relative;
		z-index: 1;
		width: 100%;
		margin: 0;
		background: transparent;
		accent-color: var(--accent);
	}

	.slider-field__meta {
		position: relative;
		z-index: 1;
		display: grid;
		grid-template-columns: auto 1fr auto;
		align-items: center;
		gap: 0.5rem;
		font-family: 'IBM Plex Mono', monospace;
		font-size: 0.8rem;
		color: var(--text-3);
	}

	.slider-field__meta strong {
		justify-self: center;
		font-size: 0.96rem;
		color: var(--text-1);
	}

	.slider-field[data-disabled='true'] {
		opacity: 0.45;
	}
</style>
