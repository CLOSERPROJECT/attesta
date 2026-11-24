/** Live Domain revisions for one Org stream. */
export type DomainEventsPort = {
	/**
	 * Calls onRevision for every revision the server sends, including the first
	 * event of each (re)connection. Returns unsubscribe.
	 */
	subscribe(stream: string, onRevision: (revision: number) => void): () => void;
};
