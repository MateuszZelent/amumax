<script lang="ts">
	import { solverState } from '$api/incoming/solver';
	import {
		postBreak,
		postFixdt,
		postMaxdt,
		postMaxerr,
		postMindt,
		postMinimize,
		postRelax,
		postRun,
		postSolverType,
		postSteps
	} from '$api/outgoing/solver';
	import { writable } from 'svelte/store';
	import Button from '$lib/ui/Button.svelte';
	import Panel from '$lib/ui/Panel.svelte';
	import ReadonlyField from '$lib/ui/ReadonlyField.svelte';
	import SelectField from '$lib/ui/SelectField.svelte';
	import TextField from '$lib/ui/TextField.svelte';

	let runSeconds = $state('1e-9');
	let runSteps = $state('100');

	const solverTypes = [
		'bw_euler',
		'euler',
		'heun',
		'rk4',
		'rk23',
		'rk45',
		'rkf56'
	].map((type) => ({ value: type, label: type }));

	function submitRun() {
		postRun(runSeconds);
	}

	function submitSteps() {
		postSteps(runSteps);
	}

	function bindNumericField<K extends 'fixdt' | 'mindt' | 'maxdt' | 'maxerr'>(
		key: K,
		submit: () => void
	) {
		return (event: Event) => {
			const next = Number((event.currentTarget as HTMLInputElement).value);
			if (!Number.isFinite(next)) {
				return;
			}

			solverState.update((state) => ({ ...state, [key]: next }));
			submit();
		};
	}
</script>

<Panel
	title="Solver"
	subtitle="Action-first runtime control with separate numerical guardrails."
	panelId="solver"
	eyebrow="Control rail"
	tone="info"
>
	<div class="solver-layout">
		<section class="solver-block">
			<header>
				<h3>Run control</h3>
				<p>Primary actions stay explicit and single-purpose.</p>
			</header>

			<SelectField
				label="Solver type"
				value={$solverState.type}
				options={solverTypes}
				onchange={postSolverType}
			/>

			<div class="solver-run">
				<TextField
					label="Runtime"
					value={runSeconds}
					unit="s"
					mono={true}
					oninput={(event) => (runSeconds = (event.currentTarget as HTMLInputElement).value)}
					onkeydown={(event) => event.key === 'Enter' && submitRun()}
				/>
				<Button variant="solid" tone="accent" onclick={submitRun}>Run</Button>
			</div>

			<div class="solver-run">
				<TextField
					label="Steps"
					value={runSteps}
					mono={true}
					oninput={(event) => (runSteps = (event.currentTarget as HTMLInputElement).value)}
					onkeydown={(event) => event.key === 'Enter' && submitSteps()}
				/>
				<Button variant="outline" tone="info" onclick={submitSteps}>Run steps</Button>
			</div>

			<div class="solver-actions">
				<Button variant="subtle" tone="accent" onclick={postRelax}>Relax</Button>
				<Button variant="subtle" tone="info" onclick={postMinimize}>Minimize</Button>
				<Button variant="outline" tone="danger" onclick={postBreak}>Break</Button>
			</div>
		</section>

		<section class="solver-block">
			<header>
				<h3>Telemetry</h3>
				<p>Readonly runtime signals separated from editable numerics.</p>
			</header>

			<div class="solver-grid">
				<ReadonlyField label="Steps" value={`${$solverState.steps}`} mono={true} />
				<ReadonlyField label="Time" value={$solverState.time.toExponential(3)} unit="s" mono={true} />
				<ReadonlyField label="dt" value={$solverState.dt.toExponential(3)} unit="s" mono={true} />
				<ReadonlyField label="Err/step" value={$solverState.errPerStep.toExponential(3)} mono={true} />
				<ReadonlyField label="Max torque" value={$solverState.maxTorque.toExponential(3)} unit="T" mono={true} />
			</div>

			<div class="solver-grid solver-grid--editable">
				<TextField
					label="Fixdt"
					type="number"
					value={$solverState.fixdt}
					unit="s"
					mono={true}
					onchange={bindNumericField('fixdt', postFixdt)}
				/>
				<TextField
					label="Mindt"
					type="number"
					value={$solverState.mindt}
					unit="s"
					mono={true}
					onchange={bindNumericField('mindt', postMindt)}
				/>
				<TextField
					label="Maxdt"
					type="number"
					value={$solverState.maxdt}
					unit="s"
					mono={true}
					onchange={bindNumericField('maxdt', postMaxdt)}
				/>
				<TextField
					label="Maxerr"
					type="number"
					value={$solverState.maxerr}
					mono={true}
					onchange={bindNumericField('maxerr', postMaxerr)}
				/>
			</div>
		</section>
	</div>
</Panel>

<style>
	.solver-layout {
		display: grid;
		gap: 1rem;
	}

	.solver-block {
		display: grid;
		gap: 0.9rem;
		padding: 0.25rem 0;
	}

	.solver-block header {
		display: flex;
		flex-direction: column;
		gap: 0.2rem;
	}

	.solver-block h3 {
		margin: 0;
		font-size: 0.95rem;
	}

	.solver-block p {
		margin: 0;
		color: var(--text-2);
		font-size: 0.88rem;
	}

	.solver-run {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 0.75rem;
		align-items: end;
	}

	.solver-actions {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.75rem;
	}

	.solver-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 0.75rem;
	}

	.solver-grid--editable {
		padding-top: 0.3rem;
	}

	@media (max-width: 767px) {
		.solver-run,
		.solver-actions,
		.solver-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
