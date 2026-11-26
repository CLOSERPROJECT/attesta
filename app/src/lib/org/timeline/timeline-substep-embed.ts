/**
 * Whether a timeline substep should host the board embed land surface
 * (EmbeddedTaskPanel / Done-band) vs the closed-state details accordion.
 *
 * After a stay Commit that drops readyWhen (e.g. last submit-many weigh),
 * orgActionable becomes false but focus / confirmRecap / just-completed pin
 * must keep the embed path so the body stays open and chrome matches.
 */
export function timelineSubstepUsesEmbed(opts: {
	hasAddress: boolean;
	orgActionable: boolean;
	focused: boolean;
	confirmRecap: boolean;
	justCompleted: boolean;
}): boolean {
	if (!opts.hasAddress) return false;
	if (opts.orgActionable) return true;
	return opts.focused || opts.confirmRecap || opts.justCompleted;
}
