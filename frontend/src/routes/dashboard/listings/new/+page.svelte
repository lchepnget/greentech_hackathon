<script lang="ts">
	import { goto } from '$app/navigation';
	import PageShell from '$lib/components/PageShell.svelte';
	import { listings } from '$lib/api';
	import { session } from '$lib/stores/session';

	let title = $state('');
	let wasteType = $state('');
	let quantity = $state('');
	let unit = $state('kg');
	let price = $state('');
	let location = $state('');
	let description = $state('');
	let files = $state<File[]>([]);
	let previews = $state<string[]>([]);
	let status = $state('');

	function choose(e: Event) {
		const input = e.currentTarget as HTMLInputElement;
		files = Array.from(input.files || []);
		previews = files.map((f) => URL.createObjectURL(f));
	}

	async function submit() {
		if (status === 'Uploading…') return;

		const missingFields = [
			!title.trim() && 'listing title',
			!wasteType && 'waste type',
			(!quantity || Number(quantity) < 1) && 'quantity',
			(!price || Number(price) < 1) && 'price',
			!location.trim() && 'location'
		].filter(Boolean);

		if (missingFields.length) {
			status = `Please complete: ${missingFields.join(', ')}.`;
			return;
		}

		status = 'Uploading…';
		const data = new FormData();
		data.append('title', title);
		data.append('wasteType', wasteType);
		data.append('quantity', quantity);
		data.append('unit', unit);
		data.append('priceSats', price);
		data.append('location', location);
		data.append('description', description);
		files.forEach((f) => data.append('photos', f));

		try {
			await listings.create({name:title,description,priceSats:Number(price)});
			await goto('/dashboard');
		} catch (e) {
			status = e instanceof Error ? e.message : 'Upload failed. Please try again.';
		}
	}
</script>

{#if $session?.role !== 'producer'}<PageShell eyebrow="PRODUCER ACCESS" title="Listing unavailable"><p class="alert">Only Producers can create listings.</p></PageShell>{:else}<PageShell eyebrow="PRODUCER LISTINGS" title="List your surplus.">
	<div class="form-card listing-form">
		<p class="state">Add clear photos so buyers can check quality before ordering.</p>
		<label>Listing title<input bind:value={title} placeholder="Fresh vegetable trimmings" required /></label>
		<label>Waste type<select bind:value={wasteType}><option value="">Choose type</option><option>Food waste</option><option>Vegetable waste</option><option>Spent grain</option><option>Poultry litter</option></select></label>
		<label>Quantity<input bind:value={quantity} type="number" min="1" /></label>
		<label>Unit<select bind:value={unit}><option>kg</option><option>bags</option><option>tonnes</option></select></label>
		<label>Price (sats / unit)<input bind:value={price} type="number" min="1" /></label>
		<label>Location<input bind:value={location} placeholder="County or town" /></label>
		<label>Description<textarea bind:value={description} rows="4" placeholder="Pickup details, freshness and storage"></textarea></label>
		<label class="upload">Listing photos<input type="file" accept="image/jpeg,image/png,image/webp" multiple onchange={choose} /><small>PNG, JPG or WebP. Multiple photos allowed.</small></label>
		{#if previews.length}
			<div class="photo-grid">
				{#each previews as src}<img src={src} alt="Selected listing preview" />{/each}
			</div>
		{/if}
		{#if status}<p class="state">{status}</p>{/if}
		<button type="button" class="btn primary" onclick={submit} disabled={status === 'Uploading…'}>
			{status === 'Uploading…' ? 'Uploading…' : 'Publish listing ↗'}
		</button>
	</div>
</PageShell>
{/if}
