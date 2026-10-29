import type { Domain } from './types.js';

export type RegisterDraftFields = {
	client: string;
	date: string;
	idoneitaRiutilizzo: string;
};

/**
 * Load / Job instance id grammar from the truck-loop fixture:
 * `{idoneitaRiutilizzo}-{YYMMDD}-{client}-{n}` where n counts matching loads + 1.
 */
export function computeIntakeInstanceId(
	domain: Domain | null | undefined,
	draft: RegisterDraftFields
): string | null {
	const client = draft.client?.trim();
	const date = draft.date?.trim();
	const idoneita = draft.idoneitaRiutilizzo?.trim();
	if (!client || !date || !idoneita) return null;
	const compact = date.replaceAll('-', '');
	if (compact.length < 6) return null;
	const yymmdd = compact.slice(2);

	let n = 1;
	if (domain?.load) {
		for (const facts of Object.values(domain.load)) {
			if (facts?.client === client && facts?.date === date) n += 1;
		}
	}
	return `${idoneita}-${yymmdd}-${client}-${n}`;
}
