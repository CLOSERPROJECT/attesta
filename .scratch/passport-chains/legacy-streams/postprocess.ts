// Usage: bun .scratch/passport-chains/legacy-streams/postprocess.ts --env <staging|production> [--in raw-export.json]
// Splits the export.js output into <_id>.yaml files, index.json, summary.md and scrub-report.txt next to the input.
// Needs Bun for Bun.YAML (pinned in mise.toml). Pure local post-processing; never talks to Mongo.
import { createHash } from "node:crypto";
import { existsSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { basename, dirname, join, resolve } from "node:path";

const here = import.meta.dir;
const args = process.argv.slice(2);
const flag = (name: string): string | undefined => {
  const i = args.indexOf(name);
  return i >= 0 ? args[i + 1] : undefined;
};

const environment = flag("--env");
if (!environment) {
  console.error("Missing --env <staging|production>; it is recorded in index.json and summary.md.");
  process.exit(2);
}
const inputPath = resolve(flag("--in") ?? join(here, "raw-export.json"));
const outDir = dirname(inputPath);
const templateDir = resolve(here, "../../../server/config");

type Raw = {
  schema: number;
  exportedAt: string;
  database: string;
  serverVersion: string;
  counts: { streamDocuments: number; streamsExported: number; processesTotal: number; processesWithDpp: number };
  streams: { _id: string; stream: unknown; fields: Record<string, unknown> }[];
  processGroups: { workflowKey: string | null; status: string | null; count: number; dppCount: number }[];
};

const raw: Raw = JSON.parse(readFileSync(inputPath, "utf8"));
if (raw.schema !== 1) throw new Error(`Unsupported export schema ${raw.schema}`);

const problems: string[] = [];

const sortKeys = (v: unknown): unknown =>
  Array.isArray(v)
    ? v.map(sortKeys)
    : v && typeof v === "object"
      ? Object.fromEntries(
          Object.entries(v as Record<string, unknown>)
            .sort(([a], [b]) => a.localeCompare(b))
            .map(([k, x]) => [k, sortKeys(x)]),
        )
      : v;

// Mirrors applyCatalogStreamCategoryAndName in server/cmd/server/seed_catalog.go: seed copies differ from the template only by these three workflow fields.
const fingerprint = (doc: any): string => {
  const copy = structuredClone(doc);
  if (copy?.workflow) {
    delete copy.workflow.categorySlug;
    delete copy.workflow.subCategorySlug;
    delete copy.workflow.name;
  }
  return JSON.stringify(sortKeys(copy));
};

const templates = new Map<string, string>();
if (existsSync(templateDir)) {
  for (const file of readdirSync(templateDir)) {
    if (!/\.ya?ml$/i.test(file) || /^categories\.ya?ml$/i.test(file)) continue;
    templates.set(fingerprint(Bun.YAML.parse(readFileSync(join(templateDir, file), "utf8"))), file);
  }
} else {
  problems.push(`Template dir ${templateDir} not found; seed-copy detection skipped.`);
}

const seededName = /\s—\s(Pilot|Standard|Extended)$/;

const groupsByKey = new Map<string, Raw["processGroups"]>();
for (const g of raw.processGroups) {
  const key = g.workflowKey ?? "(missing)";
  groupsByKey.set(key, [...(groupsByKey.get(key) ?? []), g]);
}

const instancesFor = (key: string) => {
  const rows = groupsByKey.get(key) ?? [];
  const byStatus: Record<string, number> = {};
  let total = 0;
  let dpp = 0;
  for (const r of rows) {
    const status = r.status ?? "(missing)";
    byStatus[status] = (byStatus[status] ?? 0) + r.count;
    total += r.count;
    dpp += r.dppCount;
  }
  return { total, byStatus, dpp };
};

const compressSteps = (steps: { id: string; organization: string }[]): string => {
  const runs: { org: string; ids: string[] }[] = [];
  for (const s of steps) {
    const org = s.organization || "(none)";
    const last = runs[runs.length - 1];
    if (last && last.org === org) last.ids.push(s.id);
    else runs.push({ org, ids: [s.id] });
  }
  return runs.map((r) => `${r.ids.length > 1 ? `${r.ids[0]}–${r.ids[r.ids.length - 1]}` : r.ids[0]} → ${r.org}`).join("; ");
};

const secretChecks: [string, RegExp][] = [
  ["url-credentials", /\b[a-z][a-z0-9+.-]*:\/\/[^\s/:@]+:[^\s/@]+@\S+/gi],
  ["private-key", /-----BEGIN [A-Z ]*PRIVATE KEY-----/g],
  ["secret-keyword", /\b(password|passwd|secret|api[_-]?key|access[_-]?key|private[_-]?key|authorization|bearer|credential)s?\b|\btoken\b/gi],
  ["opaque-string", /[A-Za-z0-9+/_-]{32,}={0,2}/g],
  ["url", /https?:\/\/[^\s"')]+/gi],
  ["email", /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g],
];
const mask = (s: string) => (s.length <= 8 ? "****" : `${s.slice(0, 4)}…(${s.length} chars)`);

const scrubLines: string[] = [];
const scrubCounts: Record<string, number> = {};
const scan = (id: string, source: string, text: string) => {
  text.split("\n").forEach((line, i) => {
    for (const [kind, re] of secretChecks) {
      for (const m of line.matchAll(re)) {
        scrubCounts[kind] = (scrubCounts[kind] ?? 0) + 1;
        const shown =
          kind === "url" ? m[0].replace(/\/\/[^/@\s]*@/, "//****@") : kind === "secret-keyword" ? m[0] : mask(m[0]);
        scrubLines.push(`${id}\t${source}:${i + 1}\t${kind}\t${shown}`);
      }
    }
  });
};

const index: any[] = [];
const seenIds = new Set<string>();
for (const s of raw.streams) {
  if (seenIds.has(s._id)) problems.push(`Duplicate _id ${s._id}`);
  seenIds.add(s._id);

  const yamlText = typeof s.stream === "string" ? s.stream : "";
  if (typeof s.stream !== "string") problems.push(`${s._id}: field 'stream' is ${typeof s.stream}, not a string; wrote an empty file.`);
  writeFileSync(join(outDir, `${s._id}.yaml`), yamlText);

  let parsed: any = null;
  let parseError: string | undefined;
  try {
    parsed = Bun.YAML.parse(yamlText);
  } catch (e) {
    parseError = String(e);
    problems.push(`${s._id}: YAML does not parse (${parseError})`);
  }

  const wf = parsed?.workflow ?? {};
  const steps: { id: string; title: string; order: number; organization: string }[] = (Array.isArray(wf.steps) ? wf.steps : []).map(
    (st: any) => ({
      id: String(st?.id ?? ""),
      title: String(st?.title ?? ""),
      order: Number(st?.order ?? 0),
      organization: String(st?.organization ?? ""),
    }),
  );

  const matchedTemplate = parsed ? templates.get(fingerprint(parsed)) : undefined;
  const name = String(wf.name ?? "");
  const creator = String(s.fields.createdByUserId ?? "");
  const origin = matchedTemplate
    ? `seed-copy:${matchedTemplate}`
    : seededName.test(name) && creator.startsWith("platform-admin")
      ? "seed-copy-edited"
      : "hand-authored";

  scan(s._id, `${s._id}.yaml`, yamlText);

  index.push({
    id: s._id,
    name,
    categorySlug: wf.categorySlug ?? null,
    subCategorySlug: wf.subCategorySlug ?? null,
    origin,
    fields: s.fields,
    yamlBytes: Buffer.byteLength(yamlText),
    yamlSha256: createHash("sha256").update(yamlText).digest("hex"),
    parseError,
    steps,
    organizationsDeclared: Array.isArray(parsed?.organizations) ? parsed.organizations.map((o: any) => o?.slug) : [],
    dppEnabled: parsed?.dpp?.enabled ?? null,
    instances: instancesFor(s._id),
  });
}

const orphanKeys = [...groupsByKey.keys()].filter((k) => !seenIds.has(k)).sort();
const orphans = orphanKeys.map((k) => ({ workflowKey: k, instances: instancesFor(k) }));

const groupedTotal = raw.processGroups.reduce((n, g) => n + g.count, 0);
const groupedDpp = raw.processGroups.reduce((n, g) => n + g.dppCount, 0);
const yamlFiles = readdirSync(outDir).filter((f) => /^[0-9a-f]{24}\.yaml$/.test(f));

const checks: [string, boolean][] = [
  ["streams exported == stream documents counted", raw.counts.streamsExported === raw.counts.streamDocuments],
  ["yaml files written == streams exported", index.length === raw.streams.length],
  ["no stale <_id>.yaml files from a previous run", yamlFiles.every((f) => seenIds.has(basename(f, ".yaml"))) && yamlFiles.length === seenIds.size],
  ["grouped process counts == processes total", groupedTotal === raw.counts.processesTotal],
  ["grouped DPP counts == processes with dpp", groupedDpp === raw.counts.processesWithDpp],
  ["every stream YAML parsed", index.every((i) => !i.parseError)],
];

writeFileSync(
  join(outDir, "index.json"),
  JSON.stringify(
    {
      environment,
      database: raw.database,
      serverVersion: raw.serverVersion,
      exportedAt: raw.exportedAt,
      counts: raw.counts,
      streams: index,
      orphanProcessGroups: orphans,
    },
    null,
    2,
  ) + "\n",
);

const statusText = (i: { total: number; byStatus: Record<string, number> }) =>
  i.total === 0
    ? "0"
    : `${i.total} (${Object.entries(i.byStatus)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, v]) => `${k} ${v}`)
        .join(", ")})`;
const cell = (v: string) => v.replace(/\|/g, "\\|").replace(/\n/g, " ");

const lines = [
  `Environment: ${environment}. Database: ${raw.database}. Exported at: ${raw.exportedAt}. Streams: ${raw.streams.length}. Processes: ${raw.counts.processesTotal} (${raw.counts.processesWithDpp} with a DPP).`,
  "",
  "| Stream id | Name | Steps → organization | Instances (by status) | DPP | Origin |",
  "| --- | --- | --- | --- | --- | --- |",
  ...index.map(
    (i) => `| ${i.id} | ${cell(i.name)} | ${cell(compressSteps(i.steps))} | ${statusText(i.instances)} | ${i.instances.dpp} | ${i.origin} |`,
  ),
];
if (orphans.length) {
  lines.push(
    "",
    "Process groups whose `workflowKey` matches no exported Stream (legacy disk keys, deleted Streams, or missing key):",
    "",
    "| workflowKey | Instances (by status) | DPP |",
    "| --- | --- | --- |",
    ...orphans.map((o) => `| ${o.workflowKey} | ${statusText(o.instances)} | ${o.instances.dpp} |`),
  );
}
writeFileSync(join(outDir, "summary.md"), lines.join("\n") + "\n");

writeFileSync(
  join(outDir, "scrub-report.txt"),
  [
    "# stream_id\tlocation\tkind\tmatch (opaque values and emails are masked)",
    "# Review url-credentials, private-key and secret-keyword first; urls and emails are informational.",
    ...scrubLines,
  ].join("\n") + "\n",
);

const originCounts = index.reduce<Record<string, number>>((acc, i) => ((acc[i.origin] = (acc[i.origin] ?? 0) + 1), acc), {});
console.log(`Wrote ${index.length} yaml files, index.json, summary.md, scrub-report.txt in ${outDir}`);
console.log(`Origins: ${JSON.stringify(originCounts)}`);
console.log(`Scrub findings: ${JSON.stringify(scrubCounts)}`);
for (const [label, ok] of checks) console.log(`${ok ? "PASS" : "FAIL"}  ${label}`);
for (const p of problems) console.log(`WARN  ${p}`);
if (checks.some(([, ok]) => !ok)) process.exit(1);
