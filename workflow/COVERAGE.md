# Coverage map

| Requirement | Design/spec | Implementation | Tests | Status |
| --- | --- | --- | --- | --- |
| PORT-001 | architecture/stream | exporter/importer/wire | roundtrip, V1 golden, external consumer | covered |
| PORT-002 | architecture/boundary | header and compatibility callback | rejection-before-sink, header negatives, compatibility rejection plus cancellation in both constructors | covered |
| PORT-003 | spec/limits | validation and lazy reusable copy buffers | limit boundaries, 5 MiB bounded writes, allocation matrix | covered |
| PORT-004 | architecture/stream | record digest and rolling chain | corruption, manifest, fuzz | covered |
| PORT-005 | architecture/checkpoints | checkpoint and resume constructors | codec corruption, export equivalence, full ResumeImporter rejection/cancellation/no-I/O/restoration matrix | covered |
| PORT-006 | architecture/import commit | staged sink lifecycle and bounded abort | combined staged-write/cancellation, nil-stage/cancellation, commit/abort/begin failures, stage-plus-error, cleanup deadline expiry | covered |
| PORT-007 | spec/public contracts | footer, manifest, retry, and poisoned state | empty, truncation, trailing, corrupt footer, preflight retry, partial-I/O poison | covered |
| PORT-008 | spec/errors | classified and combined errors, strict I/O contracts, explicit raw causes | public one-As aggregate causes, exact combined class sets, every-sentinel spoofing, Error-string redaction/raw-Cause tests | covered |
| PORT-009 | PRD/non-goals | API and docs | external consumer and boundary audit | covered |
| PORT-010 | distribution | `LICENSE` | license inventory | covered |

The admitted hardened source reports 94.5% statement coverage. Two fresh
independent reviews of exact evidence head `6aeecae` returned CLEAN. The exact residual
classes and why they do not leave claimed behavior untested are recorded in
`docs/verification.md`; statement percentage is not the completeness oracle.
