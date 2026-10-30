/**
 * createForm skips onSubmit when Ajv fails; leftoverTable exact-path
 * getFieldErrors misses nested progress/<id> errors. Helpers surface both.
 */

export type SubmitValidationError = {
	path: ReadonlyArray<string | number>;
	message: string;
};

export type SubmitValidationFailure = {
	errors: ReadonlyArray<SubmitValidationError>;
};

/** Messages whose path equals `parent` or is nested under it. */
export function messagesUnderPath(
	errors: ReadonlyArray<SubmitValidationError>,
	parent: ReadonlyArray<string | number>
): string[] {
	const out: string[] = [];
	for (const err of errors) {
		if (err.path.length < parent.length) continue;
		let under = true;
		for (let i = 0; i < parent.length; i++) {
			if (err.path[i] !== parent[i]) {
				under = false;
				break;
			}
		}
		if (!under) continue;
		if (err.message) out.push(err.message);
	}
	return out;
}

/** Exact-path only — what leftoverTable used before child aggregation. */
export function messagesAtExactPath(
	errors: ReadonlyArray<SubmitValidationError>,
	path: ReadonlyArray<string | number>
): string[] {
	const out: string[] = [];
	for (const err of errors) {
		if (err.path.length !== path.length) continue;
		if (path.every((p, i) => err.path[i] === p) && err.message) out.push(err.message);
	}
	return out;
}

export function formatSubmitValidationError(failure: SubmitValidationFailure): string {
	const { errors } = failure;
	if (errors.length === 0) return 'Validazione fallita';
	return errors
		.map((err) => {
			const prefix = err.path.length > 0 ? `${err.path.join('.')}: ` : '';
			return `${prefix}${err.message}`;
		})
		.join('; ');
}
