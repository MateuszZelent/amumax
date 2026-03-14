<script lang="ts">
	import StatusBadge from './StatusBadge.svelte';
	import type { ConnectionState } from './types';

	type Props = {
		state: ConnectionState;
		className?: string;
	};

	let { state, className = '' }: Props = $props();

	const tone = $derived(
		state === 'connected' ? 'success' : state === 'reconnecting' ? 'warn' : 'danger'
	);
	const label = $derived(
		state === 'connected' ? 'Connected' : state === 'reconnecting' ? 'Reconnecting' : 'Disconnected'
	);
</script>

<StatusBadge label={label} tone={tone} pulse={state !== 'connected'} className={className} />
