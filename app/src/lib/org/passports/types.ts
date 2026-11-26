/** GET /api/org/passports — Passports issued for one Org stream instance. */

export type InstancePassport = {
	subjectId: string;
	label: string;
	issuedAt?: string;
	digitalLink: string;
};

export type InstancePassportsView = {
	passports: InstancePassport[];
};
