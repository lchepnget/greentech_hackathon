<script lang="ts">
	import PageShell from '$lib/components/PageShell.svelte';
	import {wallet} from '$lib/api';
	import {session} from '$lib/stores/session';
	import type {Invoice,Transaction} from '$lib/types';

	let balance=$state<number|null>(null),tx=$state<Transaction[]>([]);
	let showWithdraw=$state(false),lnAddress=$state(''),amountSats=$state<number|null>(null);
	let showDeposit=$state(false),depositSats=$state<number|null>(null),depositInvoice=$state<Invoice|null>(null);
	let submitting=$state(false),error=$state(''),success=$state('');

	wallet.get().then(v=>balance=v.balanceSats).catch(()=>{});
	wallet.transactions().then(v=>tx=v).catch(()=>{});

	$effect(()=>{
		if(!lnAddress&&$session?.lightningAddress)lnAddress=$session.lightningAddress;
	});

	async function withdraw(){
		error='';success='';
		if(!lnAddress.trim()){error='Enter your Blink Lightning Address.';return}
		if(!amountSats||amountSats<1){error='Enter a positive amount in sats.';return}
		submitting=true;
		try{
			const result=await wallet.withdraw(lnAddress.trim(),amountSats);
			balance=balance===null?null:balance-amountSats;
			tx=[result,...tx];
			success=`Sent ${amountSats.toLocaleString()} sats to ${lnAddress.trim()}.`;
			amountSats=null;
		}catch(e){error=e instanceof Error?e.message:'Unable to complete withdrawal.'}
		finally{submitting=false}
	}

	function openBlinkWallet(bolt11:string){
		if(!bolt11)return;
		const uri=`lightning:${bolt11}`;
		window.location.assign(uri);
		setTimeout(async()=>{
			try{await navigator.clipboard?.writeText(bolt11);success='Invoice copied. Paste it into Blink if it did not open automatically.'}
			catch{success='Open Blink and paste the displayed invoice.'}
		},900);
	}

	async function deposit(){
		error='';success='';depositInvoice=null;
		if(!depositSats||depositSats<1){error='Enter a positive deposit amount in sats.';return}
		submitting=true;
		try{depositInvoice=await wallet.deposit(depositSats)}
		catch(e){error=e instanceof Error?e.message:'Unable to create Blink deposit invoice.'}
		finally{submitting=false}
	}
</script>

<PageShell eyebrow="WALLET" title="Your sats, in motion.">
	<div class="wallet-hero">
		<span>AVAILABLE BALANCE</span>
		<strong>{balance===null?'—':balance.toLocaleString()} sats</strong>
		<div>
			<button class="btn primary" onclick={()=>showDeposit=!showDeposit}>Deposit from Blink</button>
			<button class="btn ghost" onclick={()=>showWithdraw=!showWithdraw}>Send to Blink wallet</button>
		</div>
	</div>

	{#if showDeposit}
		<form class="form-card wallet-withdraw" onsubmit={(event)=>{event.preventDefault();deposit()}}>
			<p class="eyebrow">DEPOSIT FROM YOUR BLINK WALLET</p>
			<label>Deposit amount (sats)
				<input type="number" min="1" step="1" bind:value={depositSats} placeholder="10000" />
			</label>
			<button class="btn primary" disabled={submitting}>{submitting?'Creating invoice…':'Create deposit invoice ↗'}</button>
			{#if error}<p class="alert">{error}</p>{/if}
			{#if depositInvoice}
				<div class="invoice-box">
					<p class="eyebrow">PAY FROM BLINK</p>
					<img src={`https://quickchart.io/qr?text=${encodeURIComponent(depositInvoice.bolt11)}&size=180`} alt="Blink deposit invoice QR code" />
					<code>{depositInvoice.bolt11}</code>
					<strong>{depositInvoice.amountSats.toLocaleString()} sats</strong>
					<button type="button" class="btn primary" onclick={()=>openBlinkWallet(depositInvoice?.bolt11 || '')}>Open in Blink wallet ↗</button>
				</div>
			{/if}
		</form>
	{/if}

	{#if showWithdraw}
		<form class="form-card wallet-withdraw" onsubmit={(event)=>{event.preventDefault();withdraw()}}>
			<p class="eyebrow">BLINK WITHDRAWAL</p>
			<label>Blink Lightning Address
				<input type="text" bind:value={lnAddress} placeholder="username@blink.sv" autocomplete="off" />
			</label>
			<label>Amount (sats)
				<input type="number" min="1" max={balance??undefined} step="1" bind:value={amountSats} placeholder="1000" />
			</label>
			<button class="btn primary" disabled={submitting}>{submitting?'Sending…':'Withdraw to Blink ↗'}</button>
			{#if error}<p class="alert">{error}</p>{/if}
			{#if success}<p class="success">{success}</p>{/if}
		</form>
	{/if}

	<div class="form-card">
		<p class="eyebrow">TRANSACTIONS</p>
		{#if !tx.length}<p class="state">No transactions yet.</p>{/if}
		{#each tx as t}<div class="tx"><span>{t.type}</span><strong>{t.amountSats} sats</strong></div>{/each}
	</div>
</PageShell>
