<script lang="ts">
	import { solverState } from '$api/incoming/solver';
	import {
		postSolverType,
		postRun,
		postSteps,
		postRelax,
		postBreak,
		postFixdt,
		postMindt,
		postMaxdt,
		postMaxerr,
		postMinimize
	} from '$api/outgoing/solver';

	let solvertypes = ['bw_euler', 'euler', 'heun', 'rk4', 'rk23', 'rk45', 'rkf56'];
	let runSeconds = '1e-9';
	let runSteps = '100';
	let dropdownOpen = false;

	function selectSolver(type: string) {
		postSolverType(type);
		dropdownOpen = false;
	}
</script>

<section>
	<h2>Solver</h2>

	<div class="solver-grid">
		<!-- Left: Run Controls -->
		<div class="controls">
			<!-- Solver Type Dropdown -->
			<div class="dropdown-wrapper">
				<button
					class="dropdown-trigger"
					on:click={() => dropdownOpen = !dropdownOpen}
				>
					<span class="dropdown-label">Solver</span>
					<span class="dropdown-value">{$solverState.type}</span>
					<span class="dropdown-arrow">▾</span>
				</button>
				{#if dropdownOpen}
					<div class="dropdown-menu">
						{#each solvertypes as type}
							<button
								class="dropdown-item"
								class:active={$solverState.type === type}
								on:click={() => selectSolver(type)}
							>
								{type}
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Run -->
			<div class="action-row">
				<button class="action-btn" on:click={() => postRun(runSeconds)}>Run</button>
				<div class="input-with-unit">
					<input
						type="text"
						bind:value={runSeconds}
						on:change={() => postRun(runSeconds)}
						placeholder="Time"
					/>
					<span class="unit">s</span>
				</div>
			</div>

			<!-- Run Steps -->
			<div class="action-row">
				<button class="action-btn" on:click={() => postSteps(runSteps)}>Run Steps</button>
				<input
					type="text"
					bind:value={runSteps}
					on:change={() => postSteps(runSteps)}
					placeholder="Steps"
				/>
			</div>

			<!-- Relax / Minimize / Break -->
			<button class="action-btn full" on:click={postRelax}>Relax</button>
			<button class="action-btn full" on:click={postMinimize}>Minimize</button>
			<button class="action-btn full" on:click={postBreak}>Break</button>
		</div>

		<!-- Right: Numerics -->
		<div class="numerics">
			<div class="field-row">
				<span class="field-label">Steps</span>
				<span class="field-value">{$solverState.steps}</span>
				<span class="field-unit"></span>
			</div>
			<div class="field-row">
				<span class="field-label">Time</span>
				<span class="field-value">{$solverState.time.toExponential(3)}</span>
				<span class="field-unit">s</span>
			</div>
			<div class="field-row">
				<span class="field-label">dt</span>
				<span class="field-value">{$solverState.dt.toExponential(3)}</span>
				<span class="field-unit">s</span>
			</div>
			<div class="field-row">
				<span class="field-label">Err/step</span>
				<span class="field-value">{$solverState.errPerStep.toExponential(3)}</span>
				<span class="field-unit"></span>
			</div>
			<div class="field-row">
				<span class="field-label">Max Torque</span>
				<span class="field-value">{$solverState.maxTorque.toExponential(3)}</span>
				<span class="field-unit">T</span>
			</div>

			<!-- Editable fields -->
			<div class="field-row editable">
				<span class="field-label">Fixdt</span>
				<input type="number" bind:value={$solverState.fixdt} on:change={postFixdt} />
				<span class="field-unit">s</span>
			</div>
			<div class="field-row editable">
				<span class="field-label">Mindt</span>
				<input type="number" bind:value={$solverState.mindt} on:change={postMindt} />
				<span class="field-unit">s</span>
			</div>
			<div class="field-row editable">
				<span class="field-label">Maxdt</span>
				<input type="number" bind:value={$solverState.maxdt} on:change={postMaxdt} />
				<span class="field-unit">s</span>
			</div>
			<div class="field-row editable">
				<span class="field-label">MaxErr</span>
				<input type="number" bind:value={$solverState.maxerr} on:change={postMaxerr} />
				<span class="field-unit"></span>
			</div>
		</div>
	</div>
</section>

<style>
	.solver-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-lg);
	}
	@media (max-width: 800px) {
		.solver-grid {
			grid-template-columns: 1fr;
		}
	}
	.controls, .numerics {
		display: flex;
		flex-direction: column;
		gap: var(--space-sm);
	}

	/* ─── Dropdown ─────────────────────────────────────────────── */
	.dropdown-wrapper {
		position: relative;
	}
	.dropdown-trigger {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 7px 10px;
		gap: var(--space-sm);
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		color: var(--text-1);
		font-size: 13px;
		cursor: pointer;
		transition: border-color var(--duration-fast) var(--easing-default);
	}
	.dropdown-trigger:hover {
		border-color: var(--border-interactive);
	}
	.dropdown-label {
		color: var(--text-3);
		flex-shrink: 0;
	}
	.dropdown-value {
		flex: 1;
		font-weight: 600;
		font-family: var(--font-mono);
	}
	.dropdown-arrow {
		color: var(--text-3);
		font-size: 11px;
	}
	.dropdown-menu {
		position: absolute;
		top: 100%;
		left: 0;
		right: 0;
		z-index: var(--z-overlay);
		margin-top: 4px;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-lg);
		overflow: hidden;
	}
	.dropdown-item {
		display: block;
		width: 100%;
		padding: 6px 12px;
		text-align: left;
		background: transparent;
		border: none;
		color: var(--text-2);
		font-size: 13px;
		font-family: var(--font-mono);
		cursor: pointer;
		transition: all var(--duration-instant) var(--easing-default);
	}
	.dropdown-item:hover {
		background: var(--surface-3);
		color: var(--text-1);
	}
	.dropdown-item.active {
		color: var(--accent);
		font-weight: 600;
	}

	/* ─── Action buttons ──────────────────────────────────────── */
	.action-row {
		display: flex;
		align-items: center;
		gap: var(--space-sm);
	}
	.action-btn {
		padding: 7px 14px;
		background: var(--surface-2);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		color: var(--text-2);
		font-size: 13px;
		font-weight: 500;
		white-space: nowrap;
		cursor: pointer;
		transition: all var(--duration-fast) var(--easing-default);
	}
	.action-btn:hover {
		background: var(--surface-3);
		border-color: var(--border-interactive);
		color: var(--text-1);
	}
	.action-btn.full {
		width: 100%;
	}
	.input-with-unit {
		flex: 1;
		display: flex;
		align-items: center;
	}
	.input-with-unit input {
		flex: 1;
		border-radius: var(--radius-md) 0 0 var(--radius-md);
		border-right: none;
		min-width: 0;
	}
	.input-with-unit .unit {
		padding: 6px 10px;
		background: var(--surface-3);
		border: 1px solid var(--border);
		border-left: none;
		border-radius: 0 var(--radius-md) var(--radius-md) 0;
		color: var(--text-3);
		font-size: 12px;
		font-family: var(--font-mono);
	}

	/* ─── Numerics fields ─────────────────────────────────────── */
	.numerics {
		display: grid;
		grid-template-columns: auto 1fr 24px;
		gap: 0 var(--space-sm);
		align-items: center;
	}
	.field-row {
		display: contents;
	}
	.field-label {
		font-size: 12px;
		color: var(--text-3);
		padding: 6px 0;
		border-bottom: 1px solid var(--border-subtle);
		text-align: right;
		white-space: nowrap;
	}
	.field-value {
		font-family: var(--font-mono);
		font-size: 13px;
		color: var(--text-1);
		text-align: right;
		padding: 6px 0;
		border-bottom: 1px solid var(--border-subtle);
	}
	.field-unit {
		font-size: 11px;
		color: var(--text-3);
		font-family: var(--font-mono);
		padding: 6px 0;
		border-bottom: 1px solid var(--border-subtle);
	}
	.field-row.editable input {
		text-align: right;
		font-family: var(--font-mono);
		font-size: 13px;
		min-width: 0;
		border-bottom: 1px solid var(--border-subtle);
	}
</style>
