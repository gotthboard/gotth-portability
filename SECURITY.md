# Security Policy

## Supported versions

Unreleased implementation only. No version currently carries a security-support
promise. Once releases exist, supported versions will be listed here and in the
changelog. The feature branch is not a supported distribution channel.

Integrity digests are not producer authentication. Applications must
authenticate or sign archives when provenance matters, enforce authorization
before export/import, and make staged sink commits idempotent by archive ID and
sequence.

## Reporting a vulnerability

Do not disclose exploit details in a public GitHub issue, pull request,
discussion, or Forgejo ticket. Report vulnerabilities privately through
GitHub's private vulnerability-reporting form:

<https://github.com/gotthboard/gotth-portability/security/advisories/new>

Include the affected version or commit, impact, reproduction steps, and any
suggested remediation when available. The maintainers will acknowledge the
report and coordinate validation, remediation, and disclosure through the
private advisory.

Use GitHub Issues only for non-sensitive bugs. If the private form cannot be
used, open a GitHub issue containing no vulnerability details and ask the
maintainers to restore private reporting access.
