/**
 * Thin re-exports — visibility / actionable helpers live on BoardTimelineModel.
 */
export {
	actionableAddressesFromTimeline,
	actionableSeededRefsFromTimeline,
	addressVisibleOnTimeline,
	firstActionableAddress,
	recapAddressesFromTimeline,
	soleActionableAddress,
	soleActionableSeededRef,
	timelineSubstepShown,
	type SeededTaskRef,
	type TimelineShowFilter
} from './board-timeline-model.svelte.js';
