<script lang="ts">
	import { meshState as m } from '$api/incoming/mesh';
</script>

<section>
	<h2>Mesh</h2>

	<div class="mesh-grid">
		<!-- Cell Size -->
		<div class="group">
			<div class="group-label">Cell Size</div>
			<div class="field-row">
				<span class="label">dx</span>
				<span class="value">{$m.dx.toPrecision(8)}</span>
				<span class="unit">m</span>
			</div>
			<div class="field-row">
				<span class="label">dy</span>
				<span class="value">{$m.dy.toPrecision(8)}</span>
				<span class="unit">m</span>
			</div>
			<div class="field-row">
				<span class="label">dz</span>
				<span class="value">{$m.dz.toPrecision(8)}</span>
				<span class="unit">m</span>
			</div>
		</div>

		<!-- Grid Size -->
		<div class="group">
			<div class="group-label">Grid Size</div>
			<div class="field-row">
				<span class="label">Nx</span>
				<span class="value">{$m.Nx}</span>
				<span class="unit"></span>
			</div>
			<div class="field-row">
				<span class="label">Ny</span>
				<span class="value">{$m.Ny}</span>
				<span class="unit"></span>
			</div>
			<div class="field-row">
				<span class="label">Nz</span>
				<span class="value">{$m.Nz}</span>
				<span class="unit"></span>
			</div>
		</div>

		<!-- Total Size -->
		<div class="group">
			<div class="group-label">Total Size</div>
			<div class="field-row">
				<span class="label">Tx</span>
				<span class="value">{$m.Tx.toExponential(6)}</span>
				<span class="unit">m</span>
			</div>
			<div class="field-row">
				<span class="label">Ty</span>
				<span class="value">{$m.Ty.toExponential(6)}</span>
				<span class="unit">m</span>
			</div>
			<div class="field-row">
				<span class="label">Tz</span>
				<span class="value">{$m.Tz.toExponential(6)}</span>
				<span class="unit">m</span>
			</div>
		</div>

		<!-- PBC -->
		<div class="group">
			<div class="group-label">PBC</div>
			<div class="field-row">
				<span class="label">x</span>
				<span class="value">{$m.PBCx}</span>
				<span class="unit"></span>
			</div>
			<div class="field-row">
				<span class="label">y</span>
				<span class="value">{$m.PBCy}</span>
				<span class="unit"></span>
			</div>
			<div class="field-row">
				<span class="label">z</span>
				<span class="value">{$m.PBCz}</span>
				<span class="unit"></span>
			</div>
		</div>
	</div>
</section>

<style>
	.mesh-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-lg);
	}
	@media (max-width: 600px) {
		.mesh-grid { grid-template-columns: 1fr; }
	}
	.group-label {
		font-size: 10px;
		font-weight: 600;
		color: var(--text-3);
		text-transform: uppercase;
		letter-spacing: 0.1em;
		margin-bottom: var(--space-xs);
	}
	.group {
		display: grid;
		grid-template-columns: 28px 1fr 20px;
		gap: 0 var(--space-sm);
		align-items: center;
	}
	.group-label {
		grid-column: 1 / -1;
	}
	.field-row {
		display: contents;
	}
	.label {
		font-size: 12px;
		color: var(--text-3);
		font-family: var(--font-mono);
		padding: 5px 0;
		border-bottom: 1px solid var(--border-subtle);
	}
	.value {
		font-family: var(--font-mono);
		font-size: 13px;
		color: var(--text-1);
		text-align: right;
		padding: 5px 0;
		border-bottom: 1px solid var(--border-subtle);
	}
	.unit {
		font-size: 11px;
		color: var(--text-3);
		font-family: var(--font-mono);
		padding: 5px 0;
		border-bottom: 1px solid var(--border-subtle);
	}
</style>
