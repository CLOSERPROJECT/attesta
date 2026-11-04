export const QR_RESOLVE_FAILED = 'QR resolve failed';

/** Wire navigate coords from POST /api/living/qr-resolve. */
export type QrNavigate = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	groupId?: string;
	groupInstanceId?: string;
};

export type QrResolveError = {
	code: string;
	message: string;
};

/** Server result: exactly one of navigate+seed or toastable error. */
export type QrResolveResult = {
	navigate?: QrNavigate | null;
	seed?: Record<string, unknown> | null;
	error?: QrResolveError | null;
};

export type QrResolvePort = {
	resolve(raw: string): Promise<QrResolveResult>;
};

export class QrResolveHttpError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'QrResolveHttpError';
		this.status = status;
	}
}

function asRecord(raw: unknown): Record<string, unknown> | null {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
	return raw as Record<string, unknown>;
}

function parseNavigate(raw: unknown): QrNavigate | null {
	const rec = asRecord(raw);
	if (!rec) return null;
	const jobId = typeof rec.jobId === 'string' ? rec.jobId : '';
	const jobInstanceId = typeof rec.jobInstanceId === 'string' ? rec.jobInstanceId : '';
	const taskId = typeof rec.taskId === 'string' ? rec.taskId : '';
	if (!jobId || !jobInstanceId || !taskId) return null;
	const out: QrNavigate = { jobId, jobInstanceId, taskId };
	if (typeof rec.groupId === 'string' && rec.groupId) out.groupId = rec.groupId;
	if (typeof rec.groupInstanceId === 'string' && rec.groupInstanceId) {
		out.groupInstanceId = rec.groupInstanceId;
	}
	return out;
}

function parseError(raw: unknown): QrResolveError | null {
	const rec = asRecord(raw);
	if (!rec) return null;
	const code = typeof rec.code === 'string' ? rec.code : '';
	const message = typeof rec.message === 'string' ? rec.message : '';
	if (!message) return null;
	return { code: code || 'error', message };
}

function parseSeed(raw: unknown): Record<string, unknown> {
	const rec = asRecord(raw);
	return rec ? { ...rec } : {};
}

/** Normalize resolve JSON so SPA always sees toastable error or navigate+seed. */
export function parseQrResolveResult(raw: unknown): QrResolveResult {
	const body = asRecord(raw) ?? {};
	const error = parseError(body.error);
	if (error) return { error };
	const navigate = parseNavigate(body.navigate);
	if (!navigate) return { error: { code: 'invalid', message: QR_RESOLVE_FAILED } };
	return { navigate, seed: parseSeed(body.seed) };
}
