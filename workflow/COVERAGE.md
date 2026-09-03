# Coverage map

| Requirement | Design/spec | Implementation | Tests | Status |
| --- | --- | --- | --- | --- |
| PORT-001 | architecture/stream | exporter/importer/wire | roundtrip, V1 golden, external consumer | covered |
| PORT-002 | architecture/boundary | header and compatibility callback | rejection-before-sink, header negatives | covered |
| PORT-003 | spec/limits | validation and fixed copy loops | limit boundaries, 5 MiB bounded writes | covered |
| PORT-004 | architecture/stream | record digest and rolling chain | corruption, manifest, fuzz | covered |
| PORT-005 | architecture/checkpoints | checkpoint and resume constructors | codec corruption, export equivalence, import resume | covered |
| PORT-006 | architecture/import commit | staged sink lifecycle | commit/abort/write/begin failures | covered |
| PORT-007 | spec/public contracts | footer and manifest state | empty, truncation, trailing, corrupt footer | covered |
| PORT-008 | spec/errors | classified errors, distinct malformed/truncated/I/O paths | sentinel, I/O classification, and redaction tests | covered |
| PORT-009 | PRD/non-goals | API and docs | external consumer and boundary audit | covered |
| PORT-010 | distribution | `LICENSE` | license inventory | covered |

The hardened source reports 94.3% race-instrumented statement coverage. The exact residual
classes and why they do not leave claimed behavior untested are recorded in
`docs/verification.md`; statement percentage is not the completeness oracle.
