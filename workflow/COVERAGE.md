# Coverage map

| Requirement | Design/spec | Implementation | Tests | Status |
| --- | --- | --- | --- | --- |
| PORT-001 | architecture/stream | exporter/importer/wire | roundtrip, V1 golden, external consumer | covered |
| PORT-002 | architecture/boundary | header and compatibility callback | rejection-before-sink, context seams, header negatives | covered |
| PORT-003 | spec/limits | validation and lazy reusable copy buffers | limit boundaries, 5 MiB bounded writes, allocation matrix | covered |
| PORT-004 | architecture/stream | record digest and rolling chain | corruption, manifest, fuzz | covered |
| PORT-005 | architecture/checkpoints | checkpoint and resume constructors | codec corruption, export equivalence, full ResumeImporter rejection/cancellation/no-I/O/restoration matrix | covered |
| PORT-006 | architecture/import commit | staged sink lifecycle and bounded abort | commit/abort/write/begin failures, simultaneous commit failure/cancellation, stage-plus-error, pre/post-callback deadline expiry | covered |
| PORT-007 | spec/public contracts | footer, manifest, retry, and poisoned state | empty, truncation, trailing, corrupt footer, preflight retry, partial-I/O poison | covered |
| PORT-008 | spec/errors | classified errors, strict I/O contracts, explicit raw causes | every-sentinel spoofing, I/O contract, Error-string redaction/raw-Cause tests | covered |
| PORT-009 | PRD/non-goals | API and docs | external consumer and boundary audit | covered |
| PORT-010 | distribution | `LICENSE` | license inventory | covered |

The hardened source reports 94.8% race-instrumented statement coverage. The exact residual
classes and why they do not leave claimed behavior untested are recorded in
`docs/verification.md`; statement percentage is not the completeness oracle.
