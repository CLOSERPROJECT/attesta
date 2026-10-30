import jsonata, { type ExprNode } from 'jsonata';

/** ADR-0008 spirit: no non-deterministic or code-eval builtins in transforms. */
const DENIED = new Set(['now', 'random', 'eval']);

export type JsonataBindings = Record<string, unknown>;

export type JsonataSafeError =
	| { kind: 'denied'; name: string }
	| { kind: 'compile'; message: string }
	| { kind: 'evaluate'; message: string };

function collectDenied(node: ExprNode | undefined, found: Set<string>): void {
	if (!node || typeof node !== 'object') return;
	if (node.type === 'function' && node.procedure?.type === 'variable') {
		const name = String(node.procedure.value);
		if (DENIED.has(name)) found.add(name);
	}
	if (node.type === 'variable' && typeof node.value === 'string' && DENIED.has(node.value)) {
		found.add(node.value);
	}
	for (const value of Object.values(node)) {
		if (Array.isArray(value)) {
			for (const child of value) {
				if (child && typeof child === 'object' && 'type' in child) {
					collectDenied(child as ExprNode, found);
				}
			}
		} else if (value && typeof value === 'object' && 'type' in value) {
			collectDenied(value as ExprNode, found);
		}
	}
}

/**
 * `$` = input; bindings appear as `$name`.
 * Rejects `$now` / `$random` / `$eval` before running.
 */
export async function evaluateSafe(
	expression: string,
	input: unknown,
	bindings: JsonataBindings = {}
): Promise<{ ok: true; value: unknown } | { ok: false; error: JsonataSafeError }> {
	let expr: ReturnType<typeof jsonata>;
	try {
		expr = jsonata(expression);
	} catch (error) {
		return {
			ok: false,
			error: {
				kind: 'compile',
				message: error instanceof Error ? error.message : String(error)
			}
		};
	}

	const denied = new Set<string>();
	collectDenied(expr.ast(), denied);
	if (denied.size > 0) {
		const name = [...denied][0]!;
		return { ok: false, error: { kind: 'denied', name } };
	}

	try {
		const value = await expr.evaluate(input, bindings);
		return { ok: true, value };
	} catch (error) {
		return {
			ok: false,
			error: {
				kind: 'evaluate',
				message: error instanceof Error ? error.message : String(error)
			}
		};
	}
}

export function formatJsonataError(error: JsonataSafeError): string {
	if (error.kind === 'denied') return `jsonata denied: $${error.name}`;
	return `jsonata ${error.kind}: ${error.message}`;
}
