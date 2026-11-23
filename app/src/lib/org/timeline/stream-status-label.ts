import * as m from '#lib/paraglide/messages.js';

/** Paraglide label for timeline nest/substep status codes (ADR-0013). */
export function streamStatusLabel(status: string): string {
	switch (status) {
		case 'available':
			return m.stream_status_available();
		case 'pending':
			return m.stream_status_pending();
		case 'locked':
			return m.stream_status_locked();
		case 'done':
			return m.stream_status_done();
		case 'terminated':
			return m.stream_status_terminated();
		default:
			return status;
	}
}
