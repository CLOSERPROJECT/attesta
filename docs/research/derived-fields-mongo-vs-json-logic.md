# Research: Mongo expressions vs JSON Logic for Derived fields

**Ticket:** [#268](https://github.com/CLOSERPROJECT/attesta/issues/268)  
**Map:** [#267](https://github.com/CLOSERPROJECT/attesta/issues/267)  
**Domain terms:** Stream, Substep, Derived field (see `CONTEXT.md` / map #267)

## Question

Which expression language should Derived fields use as the **primary evaluation surface** — MongoDB aggregation expression language (MQL expressions), JSON Logic, or a documented hybrid/subset — given must-cover scenarios A–E and nice-to-haves F–G?

## Recommendation

**Use JSON Logic as the primary evaluation surface**, with a **small, documented Attesta custom-operator set** for gaps the stock operators do not cover (especially constructing multi-key objects for scenario B, and convenience aggregations).

Do **not** adopt Mongo aggregation expressions as the authoring/evaluation language for Derived fields. Do **not** ship a dual Mongo + JSON Logic hybrid as the public surface.

**Why (short):** #267 requires client-first live recompute and a pure function the server can re-run. JSON Logic is explicitly designed for safe, portable FE/BE evaluation of JSON-encoded rules. Mongo expressions are defined as MQL components evaluated in aggregation / `$expr` / projection contexts on the database — there is no first-party browser evaluation path. Expressiveness favors Mongo for array reshape and aggregates, but that advantage does not outweigh the portability and operational-risk failure for this product stance.

---

## Sources (primary)

| Topic | Source |
| --- | --- |
| JSON Logic intent, safety, FE/BE sharing | [jsonlogic.com](https://jsonlogic.com/) |
| JSON Logic operators (`cat`, `map`, `reduce`, `if`, `min`/`max`, …) | [jsonlogic.com/operations.html](https://jsonlogic.com/operations.html) |
| Custom operators + security caveats | [jsonlogic.com/add_operation.html](https://jsonlogic.com/add_operation.html) |
| Cross-language truthiness | [jsonlogic.com/truthy.html](https://jsonlogic.com/truthy.html) |
| Listed language parsers (JS, PHP, Python, Ruby, Go, Java, C++, …) | [jsonlogic.com/play.html](https://jsonlogic.com/play.html) |
| MQL expressions overview | [MongoDB: Expressions](https://www.mongodb.com/docs/manual/reference/mql/expressions/) |
| Expressions in aggregation pipelines | [MongoDB: Aggregation Pipeline](https://www.mongodb.com/docs/manual/core/aggregation-pipeline/) |
| `$concat` | [MongoDB: `$concat`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/concat/) |
| `$map` (incl. document-shaped `in`) | [MongoDB: `$map`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/map/) |
| `$reduce` | [MongoDB: `$reduce`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/reduce/) |
| `$size` | [MongoDB: `$size`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/size/) |
| `$sum` as array/expression | [MongoDB: `$sum` (expression)](https://www.mongodb.com/docs/manual/reference/expression/sum/) |
| `$cond` | [MongoDB: `$cond`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/cond/) |
| `$min` also as array expression (disambiguation) | [MongoDB: `$min`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/min/) |
| Server-side JS / `$function` (not a client path) | [MongoDB: Server-side JavaScript](https://www.mongodb.com/docs/manual/core/server-side-javascript/), [`$function`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/function/) |

Secondary community emulators of Mongo expressions outside `mongod` are **out of scope** for this note (not first-party specs).

---

## Constraints from the map (#267)

- Client-first live recompute while editing; server later re-runs the **same pure function**.
- Must-cover: (A) intra-Substep concat/template; (B) cross-Substep array map → `{id, boolean}` with nested override; (C) nested override as **metadata**, not expression; (D) copy-forward vs continuous bind is a **sibling** concern; (E) aggregates sum/count/min/max.
- Nice: (F) conditional derive; (G) non-form context (org/role/instance meta).
- Form shell may change; expression choice must not lock to the current Formata multi-component setup.

---

## Language snapshots

### JSON Logic

From [jsonlogic.com](https://jsonlogic.com/):

- JSON-encoded rules meant to be **shared between front-end and back-end**, stored with data, and applied to a supplied data object.
- **Not** a full programming language: no setters, no loops, no functions/gotos; “one rule leads to one decision,” no write access to host state, **never `eval()`**.
- Consistent shape: `{"operator": ["values" ...]}` (single key = operator).
- Stock ops relevant here ([operations](https://jsonlogic.com/operations.html)): `var`, `cat`, `map` / `reduce` / `filter`, `if`, arithmetic, `min` / `max`, `and` / `or`, …
- Extensible via custom ops in the JS implementation ([Adding Operations](https://jsonlogic.com/add_operation.html)); custom ops can break the “no side effects” virtue if authors allow it.
- Cross-runtime truthiness is **specified** ([Truthy and Falsy](https://jsonlogic.com/truthy.html)) so interpreters agree (e.g. empty arrays are falsy).
- Official playground lists parsers for **JavaScript, PHP, Python, Ruby, Go, Java, .NET, C++** ([play.html](https://jsonlogic.com/play.html)).

### MongoDB aggregation expressions

From [Expressions](https://www.mongodb.com/docs/manual/reference/mql/expressions/) and [Aggregation Pipeline](https://www.mongodb.com/docs/manual/core/aggregation-pipeline/):

- Expressions are **MQL components that resolve to a value** without modifying inputs.
- Documented contexts: aggregation stages (`$project`, `$addFields`, `$group`, …), query `$expr`, find projections — i.e. **database evaluation**, not an app-embedded DSL.
- Built from constants, operators (`$add`, …), and field paths (`"$path.to.field"`).
- Rich array/document operators: `$map`, `$reduce`, `$filter`, `$size`, `$concat`, `$cond`, `$sum` / `$min` / `$max` as **array expressions**, etc.
- Custom logic via `$function` / `$accumulator` runs as **server-side JavaScript** on `mongod` ([Server-side JavaScript](https://www.mongodb.com/docs/manual/core/server-side-javascript/)) — still not a browser-safe portable evaluator.

---

## Scenario coverage

| Scenario | JSON Logic (stock) | Mongo expressions | Notes |
| --- | --- | --- | --- |
| **A** Intra-Substep concat / template | `cat` + `var` | `$concat` + field paths | Both cover string concat. Neither is a full string-template language; literal glue + field refs is enough for A. |
| **B** Cross-Substep array → `{id, boolean}` | `map` yes; **multi-key object result is a gap** | `$map` with document-shaped `in` is first-class | JSON Logic rules are single-key operator objects; stock ops do not define a multi-key object constructor. Needs a **custom op** (e.g. `object` / `mergeObjects`) or host post-processing. Mongo examples build documents inside `$map.in` ([`$map`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/map/)). |
| **C** Nested override | N/A (metadata) | N/A | Per map: override allowed/disallowed is **config metadata** beside the expression, not something the language must encode. |
| **D** Copy-forward vs continuous bind | N/A (sibling) | N/A | Binding policy is outside the expression AST; either language can evaluate given a prepared data bag. |
| **E** Aggregates sum / count / min / max | Partial: `reduce`+`+` for sum; `min`/`max` over arg lists; **no stock `size`/`count`** | Strong: `$sum` (expression), `$size`, `$min`/`$max` (expression forms) | JSON Logic can compose sum via `reduce`; count/length needs custom or awkward reduce. Mongo has dedicated operators ([`$sum` expression](https://www.mongodb.com/docs/manual/reference/expression/sum/), [`$size`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/size/)). |
| **F** Conditional derive (nice) | `if` | `$cond` (also `$switch`) | Both covered. |
| **G** Non-form context (nice) | Host supplies data; `var` reads it | Host supplies document; field paths | Language-agnostic; addressing of org/role/instance meta is a **data-bag / addressing** design, not the DSL choice. |

---

## Comparison axes

### Expressiveness

- **Mongo wins on raw operators** for Attesta’s must-cover reshape/aggregate cases: document construction inside `$map`, `$size`, `$sum`/`$min`/`$max` as array expressions, `$reduce`, `$cond`.
- **JSON Logic is smaller by design** ([jsonlogic.com](https://jsonlogic.com/)): enough for decisions and transforms, but stock coverage for B and E is incomplete without composition and/or custom ops.
- For Derived fields, “enough + portable” beats “full MQL in the browser.”

### Client + server portability (browser-safe evaluation)

| | JSON Logic | Mongo expressions |
| --- | --- | --- |
| Browser | First-party JS parser; playground applies rules in-browser | No first-party in-browser expression engine; evaluation is documented as MQL/pipeline/`$expr` on the server |
| Server (Go stack) | Go parser linked from official play page | Native if evaluation is delegated to Mongo aggregation |
| Same pure function FE+BE | Design goal of the format | Only if both sides call Mongo (or a non-official reimplementation) |

**Map stance clash:** client-first live recompute cannot honestly use Mongo expressions without either (1) round-tripping every keystroke to `mongod`, or (2) maintaining an unofficial client emulator. Both are operationally worse than JSON Logic’s intended model.

### Nesting / array mapping

- Both support nested rules and `map`-style array transforms ([JSON Logic `map`](https://jsonlogic.com/operations.html), [Mongo `$map`](https://www.mongodb.com/docs/manual/reference/operator/aggregation/map/)).
- Inside JSON Logic `map`/`filter`/`reduce`, `var` is **scoped to the current element** (and `reduce` exposes `current` / `accumulator`) — parent/cross-scope access is a known awkwardness; may need custom ops or flattening the data bag before apply.
- Mongo `$map` uses named variables (`as` / `$$var`) and can emit documents in `in`.

### Operational risk

| Risk | JSON Logic | Mongo expressions as Derived-field language |
| --- | --- | --- |
| Security | No `eval()`; read-only data access by design; custom ops can reintroduce side effects if misused ([add_operation](https://jsonlogic.com/add_operation.html); method-on-object removed after prototype-pollution advisory in json-logic-js 2.0.0) | Avoid `$function`/`$where` for Derived fields; stick to pure operators if ever used server-side |
| Semantic drift FE vs BE | Mitigated by shared truthiness tests and multi-language parsers; still must pin versions and run shared fixtures | High if using a third-party client emulator vs server Mongo version |
| Authoring / support surface | One small AST + documented Attesta custom ops | Entire MQL expression surface (or an ad-hoc subset that must be inventively documented anyway) |
| Hybrid Mongo ∪ JSON Logic | — | **Worst:** two syntaxes, two truth models, two validator stories |

---

## What still needs custom helpers (either way)

Host / product concerns **outside** the expression language (still required):

1. **Data bag assembly** — current Substep form values + prior completed Substep payloads + (G) org/role/instance meta, with a stable addressing scheme.
2. **Override metadata (C)** — nested paths marked override allowed/disallowed; not encoded in the expression.
3. **Bind policy (D)** — copy-forward once vs continuous recompute; sibling to the expression.
4. **Evaluation order / dependency graph** among Derived fields on one Substep.
5. **Template sugar (A)** — optional authoring sugar that compiles to `cat` / `$concat`; not required in v1.

JSON Logic–specific custom ops to document early:

| Helper | Why |
| --- | --- |
| Multi-key **object constructor** | Scenario B: `map` → `{id, boolean}` |
| **`size` / `count`** (array length) | Scenario E; no stock length op |
| Optional **`sum` / path-aware min-max** | Ergonomics over `reduce` compositions for E |
| Optional **parent-scope `var`** | If nested `map` must read outer fields without pre-flattening |

Mongo-specific (if it had been chosen): a **full client expression runtime** (or accept server round-trips) — that helper dwarfs the JSON Logic gaps above and is not first-party.

---

## Rejected alternatives

### Mongo expressions as primary

Rejected because official evaluation contexts are database-bound ([Expressions](https://www.mongodb.com/docs/manual/reference/mql/expressions/), [Aggregation Pipeline](https://www.mongodb.com/docs/manual/core/aggregation-pipeline/)), which conflicts with client-first pure evaluation (#267). Expressiveness for B/E is real but insufficient as a deciding factor.

### Documented hybrid of both syntaxes

Rejected: doubles authoring, validation, and runtime surface without solving portability. A **JSON Logic subset + Attesta custom ops** is the useful “subset” — not Mongo∪JSON Logic.

### “Insufficient — need X”

Not needed for the language choice. Remaining open work is **data addressing**, override/bind product rules, and the small custom-op list — not another primary language bake-off.

---

## Implications for later design tickets

1. Specify the **evaluation function** as `apply(rule, dataBag) → value` using JSON Logic (+ Attesta ops), runnable in browser JS and server Go.
2. Pin interpreter versions; add **shared golden fixtures** (esp. truthiness and custom ops).
3. Keep override (C) and bind mode (D) in derivation **config**, not in the rule AST.
4. Do not couple the rule language to Formata or to executing aggregation pipelines for live UI updates.
