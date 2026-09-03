# Feature plan

| Slice | Requirements | Production units | Verification |
| --- | --- | --- | --- |
| Values and wire primitives | PORT-001/002/003/008 | types, validation, exact reads/writes | unit boundaries and fuzz |
| Export | PORT-001/003/004/005/007 | header, record, footer, checkpoint | short/long input, limits, resume |
| Import | PORT-002/004/005/006/007 | header, staged record, footer, resume | corruption, abort, unknown commit, trailing bytes |
| Persistence contract | PORT-005/008 | checkpoint codec | corruption and structural fuzz |
| Admission | PORT-009/010 | public consumer, docs, performance, review | external compile and complete gates |

Only one slice is unfinished at a time. Workflow state remains `in_progress`
through worker handoff; the orchestrator owns independent final review.
