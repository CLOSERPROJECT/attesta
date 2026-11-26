/**
 * Whether the addressed substep body should mount EmbeddedTaskPanel
 * (many land / Done-band / once-recap) vs read-only blocked/empty copy.
 *
 * Does not choose which Collapsible mounts — address always uses one shell;
 * open state lives on BoardEmbedOpen and survives readyWhen / lock flips.
 * orgActionable only gates Add / composer inside the panel.
 */
export function timelineSubstepShowsTaskPanel(opts: {
	hasAddress: boolean;
	orgActionable: boolean;
	confirmRecap: boolean;
	justCompleted: boolean;
	submissionCount: number;
}): boolean {
	if (!opts.hasAddress) return false;
	if (opts.orgActionable) return true;
	if (opts.confirmRecap || opts.justCompleted) return true;
	return opts.submissionCount > 0;
}
