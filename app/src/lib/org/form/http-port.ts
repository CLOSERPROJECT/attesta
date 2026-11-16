/** Injectable HTTP for Substep `onSubmit.http` side-effects. */

export type HttpAction = 'link' | 'download';

/** Port result: URL from strict plain-text response body; headers may carry disposition. */
export type HttpPortResponse = {
	url: string;
	headers?: Headers | Record<string, string>;
};

export type HttpRequest = {
	method: string;
	url: string;
	body?: unknown;
};

export type HttpPort = {
	request: (req: HttpRequest) => Promise<HttpPortResponse>;
};

/** Click-time helpers only — not invoked automatically on submit. */
export type HttpFollowUp = {
	openTab: (url: string) => void;
	download: (url: string, filename: string) => void;
};

/** Resolved link strip item for one successful HTTP effect. */
export type HttpLinkItem = {
	label: string;
	url: string;
	action: HttpAction;
	/** From `Content-Disposition` when `action: download`. */
	filename?: string;
};
