# Portable V1 worker evidence

## Identity and scope

- Pinned placeholder base: `64ac2f3f52f05644c1e8584b3a9044d5dda1cdd1`.
- Contract commit: `d827d1b76c7f3f6c0a3ddc965a15bfadb0fddbed`.
- Initial implementation commit: `a4b39e8d20caa140d9d5dd10fdd0e5a41ad0eff4`.
- First hardened source commit: `5a291d9752839333ba033005b21c80686836b9e3`.
- Independent-review hardening commit:
  `5ec6f2f759078fa7fe894f99ff1c245a6c67e6f4`.
- Post-repair hardening commit:
  `95edd8269173a7d580d2ab0a9ecf3560c4c2b5ce`.
- Final cancellation/provenance hardening commit:
  `d22c2de207a3a6b046b7ed96aa72d7640ca616f0`.
- Retry-boundary and cost-contract correction commit:
  `8d7836c09cf5e35c5088065c0a3906ae1ddb3e2e`.
- Final live-exporter contract qualification:
  `fa12f158ed4c0cf97a95d2cc67da5a0a1a986d40`.
- Constructor-cost and changelog-provenance correction:
  `63b7c8af4450e577d8f834d7592fcea0b32d6a3d`.
- Resume-contract, focused-coverage, and exact-file-provenance correction:
  `55ba1451cf80661f97c62cc3d41e38e2a0cc77e1`.
- Record-cost and pre-Abort deadline correction:
  `9a0392433796c26f12b1a83e04becbf9799a3c41`.
- Branch/worktree: `feature/v1-portability` at
  `/tmp/gotth-portability-worktrees/v1-portability`.
- No push, PR, tag, release, deployment, live database, secret, or remote state
  was touched.

## Boundary decisions

The package moves opaque records. Schema meaning, authorization, disclosure,
redaction, retention, legal hold, storage, transport, compression, encryption,
and release policy remain consumer-owned. Import sinks stage one record and
commit only after digest verification. Sink commit errors have unknown outcome;
consumers must reconcile and make commit idempotent by archive ID and sequence.
Cancellation observed after `Commit` is the same unknown outcome. Pre-commit
failure uses a fresh cleanup context bounded by `AbortTimeout`. A stage returned
with a Begin error is cleaned up, and cleanup deadline expiry is an `ErrSink`
failure even after a nil Abort result. Cancellation observed after a Begin error
is retained as `ErrIO` alongside the primary `ErrSink` classification. SHA-256
provides integrity, not producer authentication. `Causer.Cause` is raw and
potentially sensitive; only `Error()` is redaction-safe.

## Toolchain and capacity

- Go: `go1.26.6-X:nodwarf5 linux/amd64`.
- Graphify: 0.9.32, local code-only extraction.
- Root preflight: 5% bytes and 1% inodes used; no capacity threshold approached.
- gopls: N/A on the canonical host because the binary is absent. Direct source,
  compiler, vet, tests, coverage, fuzzing, and Graphify supplied the required
  evidence without installing or starting another tool.
- Context broker, Zoekt, and ast-grep: N/A; canonical docs and the new package
  were small enough for bounded direct reads and `/usr/bin/rg`.

## Correctness gates

Passed locally:

```text
git diff --check -- .
go vet -mod=readonly ./...
go build -mod=readonly ./...
bash /tmp/gotth-portability-changelog-audit.sh /tmp/gotth-portability-worktrees/v1-portability
make verify
go test -mod=readonly -race -count=1 -coverprofile=/tmp/gotth-portability-coverage-9a03924.out ./...
```

Hardened source coverage is 94.7% under race. Boundary, negative, staged sink,
every-byte truncation, literal empty/non-empty archive and checkpoint goldens,
resume, external-package, and bounded-write tests pass. New tests bracket every
caller-owned cancellation seam, enforce a finite no-progress bound and strict
Writer contract, verify lazy/reused payload buffers, and inject every public
sentinel through every callback class to prove causes cannot spoof
`errors.Is`. Retry-boundary tests prove every documented exporter/importer
preflight performs no record/frame I/O and leaves the live object reusable;
partial record/frame progress poisons it. Begin error paths cover nil/non-nil
stages, cancellation after the callback, and Abort success/failure.
Simultaneous begin failure and cancellation retain `ErrSink`, `ErrIO`, and both
raw causes. Both nil/error returns after a cleanup deadline have direct tests.
Zero and already-expired cleanup timeouts prove `Abort` is not invoked at all;
the fresh cleanup context is checked immediately before the callback.
Record-operation cost contracts separately account for fixed metadata, digest,
payload, and EOF I/O and distinguish local from delegated auxiliary space.
Exact-valid fuzz seeds require the commit entry even for empty payload.
`ResumeImporter` is 100% statement-covered: its argument and limit/checkpoint
ordering, entry and post-compatibility cancellation, incompatible raw-cause
redaction/classification, zero Reader I/O, and exact restoration are direct
tests. Residual lines are the defensive malformed-state, impossible-overflow,
nil-receiver, and rare delegated-I/O variants visible in the retained function
report; they are not hidden resume or compatibility gaps. The exact changelog
audit runs `git show --name-only --format=fuller` for every named commit and
compares the recorded affected-file list with the exact unique commit union.

Fresh exact-source fuzz probes ran sequentially for three seconds each:

- archive roundtrip: 132,717 executions;
- checkpoint parser: 173,839 executions;
- bounded arbitrary archive: 185,161 executions;
- valid-archive mutation oracle: 31,464 executions.

The runtime baseline `d22c2de` predates the exact-source pre-Abort deadline
guard. Its retained five-second fuzz runs remain historical evidence for the
unchanged archive format and common record paths, but do not prove the new
expired-cleanup branch:

- archive roundtrip: 19,653 executions;
- checkpoint parser: 7,696 executions;
- bounded arbitrary archive: 15,923 executions;
- valid-archive mutation oracle: 8,283 executions.

Exact commands:

```text
go test -mod=readonly -run '^$' -fuzz '^FuzzArchiveRoundTrip$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzCheckpointParserNeverPanics$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzArbitraryArchiveTerminatesWithinRecordLimit$' -fuzztime=5s ./pkg/portability
go test -mod=readonly -run '^$' -fuzz '^FuzzValidArchiveMutationOracle$' -fuzztime=5s ./pkg/portability
```

All four final runs were sequential. `ErrComplete` is not treated as a generic
successful error: exact-valid oracles require one commit, map membership, exact
bytes, zero aborts, counts, digest-derived rolling chain, Manifest, and EOF.
Invalid mutation classes assert no forbidden commit, completion, or Manifest.

## External consumer

A separate temporary module imported the public package from a no-local clone
detached at runtime baseline `d22c2de` and passed race, vet, and build. It
constructs exporter/importer, persists/parses a checkpoint, implements the
staged sink interfaces, and observes explicit completion. The no-local clone
itself passed readonly race, vet, and build with a clean detached status. Both
temporary directories are disposable. Every final trace records exact detached
HEAD and clean porcelain-v2 status, source hashes, toolchain/kernel, literal
command, start/end timestamps, explicit exit status, SHA-256 of a separately
retained raw output, and that raw output inline. The clean-clone proof also
records explicit per-command exits for race/vet/build. The external trace prints
and hashes its module files, hashes the replaced source, resolves `go.mod`
replacement with `go list -m -json all`, and records explicit per-command exits
for race/vet/build. Trace helpers, traces, and raw artifacts are retained and
hashed below, so helper-backed command records remain reproducible.

## Performance evidence

The `d22c2de` end-to-end matrix and limitations are in
`docs/performance.md`. Zero-record
export uses 304 B/op and three allocations; one empty record uses 440 B/op and
nine allocations. Zero-record import uses 544 B/op and 21 allocations; one
empty record uses 776 B/op and 35 allocations. Non-empty export/import remains
about 33.2/33.5 KiB per operation from 1 KiB through 16 MiB payloads. Thus empty
records do not pay the 32 KiB buffer, and the first non-empty record's buffer is
reused. No runtime speedup is claimed; Amdahl inputs are N/A. The exact-source
runtime change only suppresses an Abort callback when the fresh cleanup context
is already expired. It has direct correctness coverage; no performance claim
or baseline reuse is made for that branch.

Exact uninstrumented commands were:

```text
env GOTTH_PORTABILITY_PERF=1 go test -mod=readonly -run '^TestPerformanceSamples$' -count=1 -v ./pkg/portability
go test -mod=readonly -run '^$' -bench '^Benchmark(Export|Import)$' -benchmem -benchtime=500ms -count=5 ./pkg/portability
```

The harness reported `GOMAXPROCS=4`; each export benchmark operation is serial.

Worker review also split `ErrIO` from `ErrMalformed` and `ErrTruncated`.
Independent review then found that callback causes could counterfeit public
sentinels. `errors.Is` now exposes only the library classification. `Causer`
provides explicit access to the unchanged raw cause, which may be sensitive and
must be consumer-redacted before logging or display; public `Error()` text
remains redaction-safe.

| Exact source `9a03924` retained artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-9a03924.log` | `2ae85e9708dfb33d94a4fb1efa387b01f446691b25ad3d5c8e40a532b9a4ed1a` |
| `/tmp/gotth-portability-verify-9a03924.raw.log` | `17654113acdc9fbc2a6b659b8a2ca63f89a27ed2e12f62a03754df1856a102f4` |
| `/tmp/gotth-portability-coverage-9a03924.log` | `de63f246b9b910fb05f7584a68a5ebb312d635b63fb8cb362ec127618d9492b8` |
| `/tmp/gotth-portability-coverage-9a03924.raw.log` | `69cd464fe4f5316675f339c3adeadd9140f6dcc089b064283ce9dc3890d45ad1` |
| `/tmp/gotth-portability-coverage-9a03924.out` | `2c84d8453269f3c727dfcc1998b000f8b960d50e4cf0319b6e054e4498be2705` |
| `/tmp/gotth-portability-coverage-functions-9a03924.log` | `61abce5e0517e2d6a0d6c61def7f94f3bd7be6e69f38efca5464c64f49a89f46` |
| `/tmp/gotth-portability-coverage-functions-9a03924.raw.log` | `adba68a3c8da10fa60b9f0e8eb44a39deda31ee277b6d772a6beebba8b8e16a1` |
| `/tmp/gotth-portability-focused-race-1-9a03924.log` | `c448aaf78a180e291864200f0e89491fa0bd570dfe24f23502514cbbf70384f1` |
| `/tmp/gotth-portability-focused-race-1-9a03924.raw.log` | `ed891fd0d40ea61433bfc3f9b5c6ece9ca69bc466de196c111e97c50a9f84240` |
| `/tmp/gotth-portability-focused-race-2-9a03924.log` | `ff10f858271198ae654f65b8f06b784ea2acbfa22a511e5c4d51474e0bb4b0af` |
| `/tmp/gotth-portability-focused-race-2-9a03924.raw.log` | `df2e7876e667aac61d57d7cf59d9352d0e4e7f7218c74d1644b7d529e17c7b41` |
| `/tmp/gotth-portability-focused-race-3-9a03924.log` | `929c63e2a8395a6e2294bbb1d3216ada3d3ee01df53d2fcd285530103053f0c7` |
| `/tmp/gotth-portability-focused-race-3-9a03924.raw.log` | `ed891fd0d40ea61433bfc3f9b5c6ece9ca69bc466de196c111e97c50a9f84240` |
| `/tmp/gotth-portability-fuzz-roundtrip-9a03924.log` | `fa6377555c216c58b667e91f92c2dc09d2f2ccfcf688a93b695cdf102b9c0854` |
| `/tmp/gotth-portability-fuzz-roundtrip-9a03924.raw.log` | `7365b690d1eecc0296fa01b00a272ddd4093ce28bc5265d4e4e36f9eedf284b1` |
| `/tmp/gotth-portability-fuzz-checkpoint-9a03924.log` | `cf77797d6ae9eec074e97c29555436f5544fe843b5a2aebe4f4ac6679455be43` |
| `/tmp/gotth-portability-fuzz-checkpoint-9a03924.raw.log` | `84eed6c0206ce0625c62b2b77b418d9396d97291d661188c2073449ef57c3f06` |
| `/tmp/gotth-portability-fuzz-arbitrary-9a03924.log` | `745cd7a749e1bbb15118b496ac4af80a265fcd155d533ef4e7128f39258679b1` |
| `/tmp/gotth-portability-fuzz-arbitrary-9a03924.raw.log` | `7fccba8f95c739ac6e2e42597352fdcd18ad4f9625684c3008df9da587e8ed22` |
| `/tmp/gotth-portability-fuzz-mutation-9a03924.log` | `89bc668e33e40f6f38f9373be44861de0a147046ccdc3e69017d4c1054083bca` |
| `/tmp/gotth-portability-fuzz-mutation-9a03924.raw.log` | `7d4acd0b42e54a23a783eefa2cb1428aef051a5214c9289335e36e57d405ad04` |
| `/tmp/gotth-portability-clean-clone-9a03924.log` | `746feb0b19c6782e391643ac1e0f21a22a4d17b1d81c2dff8a3b7e9730ea5c94` |
| `/tmp/gotth-portability-clean-clone-9a03924.raw.log` | `ceeb18140aa071d2897ce3675102d23dd4ce013c1dcb6a7a683172a8a7d36c81` |
| `/tmp/gotth-portability-evidence-9a03924.sh` | `2ddee750d65dbdf0058f47d52548f6f8ec84f2263e0159a6387c57f50d5a6a16` |

| Exact source `55ba145` retained artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-55ba145.log` | `8804f737678f19e0c11c4c92f6a5634593ee9c0c55a14a9b742a88190bc09955` |
| `/tmp/gotth-portability-verify-55ba145.raw.log` | `17654113acdc9fbc2a6b659b8a2ca63f89a27ed2e12f62a03754df1856a102f4` |
| `/tmp/gotth-portability-coverage-55ba145.log` | `495c3332d2c04156126948505331b74b2f6d3a9f87460bcfe19ebeeeec1c102d` |
| `/tmp/gotth-portability-coverage-55ba145.raw.log` | `e371950355e510d53310c23e41a555417ec27479640f91f82076b8d62fd99062` |
| `/tmp/gotth-portability-coverage-55ba145.out` | `ee5b4b8ccc998042050b1d2a43a6eb1201e55570bb37c7046e2f9b64a7c4d4b7` |
| `/tmp/gotth-portability-coverage-functions-55ba145.log` | `a23b9dcd1e3776dcc17a19dc9230534b5527d76c70dd560b0040955b9b2c4bc4` |
| `/tmp/gotth-portability-coverage-functions-55ba145.raw.log` | `679548d6defb6fc7ed4694f8ded58acc65e4dd195ec645653673da52a9349032` |
| `/tmp/gotth-portability-focused-race-1-55ba145.log` | `23544ab801af28bc23dcc62d5bc865bca6a91c1243c0ad37cc6458c69930f5ef` |
| `/tmp/gotth-portability-focused-race-1-55ba145.raw.log` | `6b8e8c43225fbe614b5ca8e7ea21fef17c1666a0ffdf86cd0680cbfd726093bc` |
| `/tmp/gotth-portability-focused-race-2-55ba145.log` | `4357fcdbe73c6f74d6f767ccf45c49bb9ccf19043a716ff4a848f70a8f5e9ccc` |
| `/tmp/gotth-portability-focused-race-2-55ba145.raw.log` | `ba8f02d937482fbc3ec48c49e60f876ba7165e17148ffc3eb1e559106a34994e` |
| `/tmp/gotth-portability-focused-race-3-55ba145.log` | `458631de199686f5df58b2bd190c3fdd070cd4e061f9f9d68ff4e46118f5fbd9` |
| `/tmp/gotth-portability-focused-race-3-55ba145.raw.log` | `ba8f02d937482fbc3ec48c49e60f876ba7165e17148ffc3eb1e559106a34994e` |
| `/tmp/gotth-portability-fuzz-roundtrip-55ba145.log` | `2777f3e692cdc9437008e562e401e3f01ddd1176b035db62d57c36b0a0d5bacd` |
| `/tmp/gotth-portability-fuzz-roundtrip-55ba145.raw.log` | `77a7021cb82a63d47b2420aa9898021cf0e7168a51a80264c62d1b6a80e55f32` |
| `/tmp/gotth-portability-fuzz-checkpoint-55ba145.log` | `9ea404bf6fe25542b11d1dbf08ebf7c4fee7929d6c7f37a0c64edd8595e394da` |
| `/tmp/gotth-portability-fuzz-checkpoint-55ba145.raw.log` | `bfcd39a8b2ef7cbb593f713d031dab0cacfcad0cd5b6cc76f9a43e7ad8522ed4` |
| `/tmp/gotth-portability-fuzz-arbitrary-55ba145.log` | `44d8c3997b212a789d9fdf06a08301df98c1fb8e9807653b997470bf3ee7d953` |
| `/tmp/gotth-portability-fuzz-arbitrary-55ba145.raw.log` | `7771b7171f61c8c5cc1538df80180a2f64c1b2b6103943222f9eb30264d1ce0c` |
| `/tmp/gotth-portability-fuzz-mutation-55ba145.log` | `2708631398deaa5c8c99fde5430bb483139c5f29cacecb24ebb82121aa248e36` |
| `/tmp/gotth-portability-fuzz-mutation-55ba145.raw.log` | `9596fa41a5962de8734652299ceedd0bf5b2c54161489d8481b7391069350ca2` |
| `/tmp/gotth-portability-clean-clone-55ba145.log` | `cd8425eb2f478538c133f34415f954f81a011b3aaae1b4822426f04886197560` |
| `/tmp/gotth-portability-clean-clone-55ba145.raw.log` | `c7ef7e74144a76692188d478ee855f6094dcdeab59855e5602bda21a973794a4` |
| `/tmp/gotth-portability-evidence-55ba145.sh` | `c3cf48e001529f0577361f3981e2d285e8eb96384295dfcadfa89155f46d7110` |
| `/tmp/gotth-portability-changelog-audit.sh` | `da134c5d0b493d2f9ccc8b9b1ec3710b8c207bc27e277a1261a19864c16d3318` |

| Prior contract-audit artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-contract-63b7c8a.log` | `532850d2138a1aa4c57ab0d64bf43ab4286febe6ead4dfbbf3be10fcf07f2956` |
| `/tmp/gotth-portability-contract-63b7c8a.raw.log` | `203875350dc9b04020cfec372572691f9eb8f4b2dd6edcea13b4448d9b9129b7` |
| `/tmp/gotth-portability-contract-check-63b7c8a.sh` | `0881ea0fc3844e6a32f11839528ab1f8afc7344875587420d5aaf25e12f91a60` |
| `/tmp/gotth-portability-verify-63b7c8a.log` | `226f3c85bd20fb608ea08b563794974e9211a8f61c7e001f633b27496cc47807` |
| `/tmp/gotth-portability-verify-63b7c8a.raw.log` | `00f07a4e6e0deed70a4f75444b225341e4e798262a485303e3ace64502877e3d` |
| `/tmp/gotth-portability-coverage-63b7c8a.log` | `1773d73c920eba02285aaffa63469a4bafbe1b9402a60b4e094b0c954dcbc959` |
| `/tmp/gotth-portability-coverage-63b7c8a.raw.log` | `7b2575b6905bb690573d9cade464e6543211ff792b404054475696084f89e2d3` |
| `/tmp/gotth-portability-coverage-63b7c8a.out` | `2b3dd68f02583c8ad3d6ccb4a0eaf9f32ac3a01eeea0828dd2cca46d8e52ec11` |

The following artifacts bind runtime baseline `d22c2de`. They remain relevant
only to unchanged archive-format, common-path performance, external-consumer,
and historical graph claims. They do not prove the later exact-source
pre-Abort expired-context branch and are not represented as current-source
executions.

| Unchanged-runtime baseline artifact | SHA-256 |
| --- | --- |
| `/tmp/gotth-portability-verify-d22c2de.log` | `3d424f8690a0d1a835cf5de1620e58904c42ae8a94642d8ab2051b56b47b7507` |
| `/tmp/gotth-portability-verify-d22c2de.raw.log` | `5d4014153793a187de23270b75de8a100fbb4170d4951d5b717abee699a89af7` |
| `/tmp/gotth-portability-coverage-d22c2de.log` | `63668c9601c257d818d38811fc0510c102cd7e86c09c36d72fd468ba336c7674` |
| `/tmp/gotth-portability-coverage-d22c2de.raw.log` | `7267971e0853a381a1e1e1a64fca7d2413dac7910c55a516d6c5b7c98f240a97` |
| `/tmp/gotth-portability-coverage-d22c2de.out` | `162437dd10f8e8e50da8c3154c3b86d996ae2dccae23e41dbdc4f80ba43a2f49` |
| `/tmp/gotth-portability-race50-d22c2de.log` | `a4dc7d8c3ad85246b68be01d11a2a84d7efbe37f2ec2b442efe7ff59e31a4391` |
| `/tmp/gotth-portability-race50-d22c2de.raw.log` | `fab3a1c9685109b80455e816238da73706c380824898f1e3ac62db1949ad6db7` |
| `/tmp/gotth-portability-fuzz-roundtrip-d22c2de.log` | `45b4a6e6a0a3b7bd2a146391ebb7b984c9331c8feae1dfe24f97766330a13f32` |
| `/tmp/gotth-portability-fuzz-roundtrip-d22c2de.raw.log` | `47e86cd1fde75a48a1d5e7cfff9d292cfcf3683a9c937f7975c845223861df23` |
| `/tmp/gotth-portability-fuzz-checkpoint-d22c2de.log` | `2ef4bc921341014da286cc78f134c250cf28ad18de4ac905333424aea79ebca8` |
| `/tmp/gotth-portability-fuzz-checkpoint-d22c2de.raw.log` | `49929053895723b42a71c0911e03f7e0eefa5e0c3607b592e7865c2ea6c13dfb` |
| `/tmp/gotth-portability-fuzz-arbitrary-d22c2de.log` | `3ef65f4045f13706edab166bb71da961e3402c209454ba34b87ae20644b0c448` |
| `/tmp/gotth-portability-fuzz-arbitrary-d22c2de.raw.log` | `43425cbcd02802589d102d3dfe500f5a743d13f6a253eefb2038854be5b08a55` |
| `/tmp/gotth-portability-fuzz-mutation-d22c2de.log` | `abee5b0d8369fefb85c92ba97cfe498082070a5ad2c7dde0c192618f13f70d3c` |
| `/tmp/gotth-portability-fuzz-mutation-d22c2de.raw.log` | `c771cdb1f21923eed3cbc1b4f69368b7ae4d8687c21b78082754f1e20c300fb2` |
| `/tmp/gotth-portability-performance-d22c2de.log` | `43cdf0a96f9c4bc0800ef906309bcd997ea0dee53d9a8af9f91c0fcd39a7b904` |
| `/tmp/gotth-portability-performance-d22c2de.raw.log` | `0a33cad1e03f7867d58e5cc51f3d123e350ac9c73a00dd84ca36ba73135776e5` |
| `/tmp/gotth-portability-bench-d22c2de.log` | `b970b5f0e07d0fb67e5165e699c05babccaddd1b272a07ed85de2713790e4eac` |
| `/tmp/gotth-portability-bench-d22c2de.raw.log` | `adbce8469f5f916a834d706bd233de48427c24b97e7aacd9c67f098bb2bece38` |
| `/tmp/gotth-portability-clean-clone-d22c2de.log` | `1c0c04cd6664714b6e8a694ea140228baf16e5d5cd08fab2864efc0b4fdf36b0` |
| `/tmp/gotth-portability-clean-clone-d22c2de.raw.log` | `5c7e0b44950f4b2cad4ac261d962368092554dd29f62879f57f81db73924255c` |
| `/tmp/gotth-portability-external-proof-d22c2de.log` | `491fd334c59ba2fa4d57dd94bb8713127aecaf0947b446328259733d83142151` |
| `/tmp/gotth-portability-external-proof-d22c2de.raw.log` | `90c2e6130546e464e06b0c367b0154359415673aefd8bc90f676a369b077bf8a` |
| `/tmp/gotth-portability-graphify-d22c2de.log` | `825139aa916058b7eed425c227bdc3a4a28cd90cb98bb6dbf13734857ebe498f` |
| `/tmp/gotth-portability-graphify-d22c2de.raw.log` | `5f92c7ac39d3719e08cc5f1d42375003c86c05b2900f55ce51eac7ba66745419` |
| `/tmp/gotth-portability-graphify-cluster-d22c2de.log` | `718ee77405e17ac23408f69a14e9256d4160ac01c8906a12a63a09628887a212` |
| `/tmp/gotth-portability-graphify-cluster-d22c2de.raw.log` | `dc360fc38197f6072e16f61e675c0a487129153561b17818528d3f291cefbfdc` |
| `/tmp/gotth-portability-graph-audit-d22c2de.log` | `2d190674adf087d6ee8f5ba374858a8622a0b7573e1e5b105865a70d8b2909c9` |
| `/tmp/gotth-portability-graph-audit-d22c2de.raw.log` | `bcde7a87b8201fb6e213b3296222788b5ada910e74efeacb407890512a1f6bdb` |
| `/tmp/gotth-portability-trace-d22c2de.sh` | `cbd5361acdad7e1ce8f742f22f5622426e9d9955bdadf477f3b9e65a528d5ef9` |
| `/tmp/gotth-portability-clean-run-d22c2de.sh` | `99edd5829dda4f8d4ed1b121861c6faabb4f7ea28cc02e88e5a984d6e177e21f` |
| `/tmp/gotth-portability-external-run-d22c2de.sh` | `38c7372099dd304441d374e7aff1bba18706651034d714ec0c70a2f9c6f75720` |
| `/tmp/gotth-portability-graph-audit-d22c2de.sh` | `da70f829c463a40a752aaeaab3d4fe5411b7199410721cd2b81cd598bfc28134` |
| `/home/linus/.cache/openclaw-graphify/gotth-portability/d22c2de207a3a6b046b7ed96aa72d7640ca616f0/graphify-out/graph.json` | `f5898d27c61228193a62247126aefeba46fe250e756f49c19dee951fc45b40bc` |
| `/home/linus/.cache/openclaw-graphify/gotth-portability/d22c2de207a3a6b046b7ed96aa72d7640ca616f0/graphify-out/GRAPH_REPORT.md` | `cd2126d3280a0e7f40a1823b84ff3f3139b3f83b6f09fa65bd6502018a94a34c` |

## Graph review

Graphify extracted hardened source commit
`d22c2de207a3a6b046b7ed96aa72d7640ca616f0` in code-only mode: 274 nodes,
677 edges, and 14 communities. It skipped 17 non-code documents and six
unclassified non-code files. The graph is at
`~/.cache/openclaw-graphify/gotth-portability/d22c2de207a3a6b046b7ed96aa72d7640ca616f0/graphify-out/graph.json`
with SHA-256
`f5898d27c61228193a62247126aefeba46fe250e756f49c19dee951fc45b40bc`.
The report records built-from commit `d22c2de2`; extraction, clustering, and
audit traces independently bind exact detached source and record raw outputs.
Graph and source checks cover the Begin/Abort path and exact-valid fuzz helper;
ambiguous method names were resolved directly in source. The graph contains no
self-loop or exact duplicate edge. Graph output is iteration evidence, not
correctness or admission.
The graph predates `9a03924` and therefore does not model its one new deadline
guard. That localized branch was reviewed and tested directly; no exact-source
graph claim is made for this repair.

## Cold review

Earlier worker reviews corrected checkpoint invariants, first-record recovery,
unknown-commit modeling, I/O classification, wire/boundary pins, and evidence.
Independent review of exact prior head `66feed0` then rejected cancellation
boundaries, callback error identity, Reader/Writer contract handling,
per-record 32 KiB allocation, and a weak arbitrary-archive fuzz oracle. Those
implementation and evidence findings are corrected in exact source `5ec6f2f`.
Independent review of documentation/evidence head `ca9de4d` then found raw
`Cause` mislabeled as redacted, abandoned Begin stage-plus-error state, missing
post-Abort deadline reporting, missing cost comments, empty-payload fuzz oracle
holes, silent proof logs, and stale changelog chronology. All technical
findings are corrected in exact source `95edd82`; the final evidence commit is
documentation-only.
Independent review of evidence head `e4fe93f` then found Begin callback
cancellation suppression, understated checkpoint/resume temporary space, a
false unconditional ParseCheckpoint lower bound, unbound raw admission logs,
and missing changelog times. Implementation `d22c2de` corrects the callback and
complexity findings. The provenance-bound traces and this documentation-only
evidence commit correct the evidence and chronology findings.
Independent review of evidence head `d2bd5ea` then rejected an overbroad
poisoning claim and invalid general lower bounds. Exact source `fa12f15`
documents the actual retry/progress boundary, corrects the bounds, and adds
direct state-transition tests. Fresh focused, full, and coverage traces bind
that correction; unchanged-runtime fuzz, performance, external-consumer, and
graph results remain explicitly labeled as the `d22c2de` baseline.
Independent review of evidence head `c510f4b` then found the same false general
lower-bound class in `NewExporter` and seven historical changelog headings that
contradicted their named commits' Git times. Exact source `63b7c8a` corrects the
constructor contract, normalizes and orders the headings, and defines their
author-time provenance. A retained focused audit checks both claims against
source and Git history; full and coverage traces bind the exact correction.
Independent review of evidence head `b1f718a` then found five adjacent cost
contracts that still counted skipped delegated calls, historical changelog file
claims that did not equal their named commits, and untested public
`ResumeImporter` rejection/cancellation branches. Exact source `55ba145`
qualifies each named cost path, replaces every affected-file shorthand with the
exact named-commit union, and raises `ResumeImporter` to 100% statement
coverage with direct no-I/O and restoration proof. Fresh exact-source traces
bind full verification, 94.7% race coverage, focused race x3, four fuzz probes,
and a clean clone. The final evidence commit remains documentation-only.
Independent review of evidence head `88d2c36` then found that record-operation
cost contracts omitted fixed metadata/digest I/O and delegated auxiliary space,
and that `abortWithTimeout` could call `Abort` even when its fresh cleanup
context had already expired. Exact source `9a03924` accounts for each I/O and
space component, checks the fresh context immediately before the callback, and
adds an expected-red/green zero-callback regression test for zero and negative
timeouts. Fresh exact-source traces bind full verification, 94.7% race
coverage, focused race x3, four fuzz probes, and a clean detached clone. This
final evidence commit remains documentation-only.

## Remaining gate

No actual downstream consumer schema or dependency pin exists. Inventing one
would defeat portability's consumer-owned boundary. That product integration
is the remaining blocker; workflow stays `in_progress`, and the package remains
unreleased. The orchestrator owns any further independent review and admission.
