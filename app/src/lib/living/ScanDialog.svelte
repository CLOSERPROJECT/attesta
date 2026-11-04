<script lang="ts">
	import Scan from '@lucide/svelte/icons/scan';
	import { CameraError, type QrResult } from 'cam2qr';
	import { createQrScanner } from 'cam2qr/svelte';

	import { Button } from '#lib/shadcn/ui/button/index.js';
	import * as Dialog from '#lib/shadcn/ui/dialog/index.js';

	type Props = {
		open: boolean;
		busy?: boolean;
		onOpenChange: (open: boolean) => void;
		onScan: (raw: string) => void | Promise<void>;
	};

	let { open, busy = false, onOpenChange, onScan }: Props = $props();

	let manual = $state('');
	let handling = false;

	function hintForCameraError(err: CameraError): string {
		if (err.code === 'permission-denied') {
			return 'Camera permission denied — paste the code below.';
		}
		return 'Camera not available — paste the code below.';
	}

	async function acceptScan(raw: string) {
		if (!raw || busy || handling) return;
		handling = true;
		try {
			manual = '';
			onOpenChange(false);
			await onScan(raw);
		} finally {
			handling = false;
		}
	}

	const { video, error, isScanning } = createQrScanner({
		camera: { facing: 'environment' },
		stopOnDecode: true,
		onDecode(r: QrResult) {
			void acceptScan(r.text.trim());
		}
	});

	// Hide stale errors once the stream is up; paste stays available either way.
	let camError = $derived.by(() => {
		if (!open || $isScanning) return null;
		const err = $error;
		if (!(err instanceof CameraError)) return null;
		return hintForCameraError(err);
	});

	async function submitManual() {
		const raw = manual.trim();
		if (!raw || busy || handling) return;
		await acceptScan(raw);
	}

	function handleOpenChange(next: boolean) {
		if (!next) manual = '';
		onOpenChange(next);
	}
</script>

<Dialog.Root {open} onOpenChange={handleOpenChange}>
	<Dialog.Content data-testid="scan-dialog" class="sm:max-w-md">
		<Dialog.Header>
			<Dialog.Title>Scan</Dialog.Title>
			<Dialog.Description>
				Point at a label QR, or paste the link / id.
			</Dialog.Description>
		</Dialog.Header>

		<div class="space-y-3">
			<div
				class="aspect-video overflow-hidden rounded-md border border-zinc-200 bg-zinc-950"
				data-testid="scan-camera"
			>
				{#if open}
					<video
						class="size-full object-cover"
						use:video
						autoplay
						muted
						playsinline
					></video>
				{/if}
			</div>
			{#if camError}
				<p class="text-xs text-zinc-500" data-testid="scan-camera-hint">{camError}</p>
			{/if}

			<form
				class="flex gap-2"
				onsubmit={(e) => {
					e.preventDefault();
					void submitManual();
				}}
			>
				<input
					class="min-w-0 flex-1 rounded-md border border-zinc-200 px-3 py-2 text-sm"
					data-testid="scan-manual-input"
					placeholder="Paste entityId or URL"
					autocomplete="off"
					bind:value={manual}
					disabled={busy}
				/>
				<Button
					type="submit"
					data-testid="scan-manual-submit"
					disabled={busy || !manual.trim()}
				>
					{#if busy}
						…
					{:else}
						<Scan class="size-4" />
						Go
					{/if}
				</Button>
			</form>
		</div>
	</Dialog.Content>
</Dialog.Root>
