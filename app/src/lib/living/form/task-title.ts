/** Prefer authored schema.title for Task chrome; fall back to task id. */
export function taskTitleFromConfig(authored: unknown, taskId: string): string {
	if (authored !== null && typeof authored === 'object' && !Array.isArray(authored)) {
		const schema = (authored as Record<string, unknown>).schema;
		if (schema !== null && typeof schema === 'object' && !Array.isArray(schema)) {
			const title = (schema as Record<string, unknown>).title;
			if (typeof title === 'string' && title.length > 0) return title;
		}
	}
	return taskId;
}
