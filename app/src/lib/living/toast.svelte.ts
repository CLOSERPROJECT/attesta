/** Ephemeral operator toast (QR resolve errors, etc.). */
class LivingToastStore {
	message = $state.raw<string | null>(null);
	#clearTimer: ReturnType<typeof setTimeout> | null = null;

	show = (message: string, ttlMs = 5000): void => {
		const text = message.trim();
		if (!text) return;
		if (this.#clearTimer) {
			clearTimeout(this.#clearTimer);
			this.#clearTimer = null;
		}
		this.message = text;
		this.#clearTimer = setTimeout(() => {
			this.message = null;
			this.#clearTimer = null;
		}, ttlMs);
	};

	dismiss = (): void => {
		if (this.#clearTimer) {
			clearTimeout(this.#clearTimer);
			this.#clearTimer = null;
		}
		this.message = null;
	};
}

export const livingToast = new LivingToastStore();
