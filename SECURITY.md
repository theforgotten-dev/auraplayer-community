# Security policy

Security fixes target the current Community source on `dev` and the newest
Community release. Older preview builds may contain vulnerabilities fixed later;
update to the newest release and consult its notes.

## Report a vulnerability privately

Use [GitHub private vulnerability reporting](https://github.com/theforgotten-dev/auraplayer-community/security/advisories/new).
Include the affected version, impact and minimal reproduction. Do not post
credentials, personal information or an unpatched exploit in a public issue.
Do not send unsolicited executable attachments.

The project does not promise a fixed response time or offer a bug bounty.

## Automated controls

Go tests, vet and govulncheck run on pushes and pull requests to `dev`, and
weekly. Gitleaks scans Git history with redacted output. CodeQL scans Go code.
Dependabot checks Go modules and GitHub Actions weekly. Actions use pinned
commit hashes and least-privilege workflow tokens.

These checks do not constitute an audit of all C/C++ dependencies bundled in
libmpv/FFmpeg. The native runtime is pinned by URL and SHA-256 in the release
script. Review its provenance and native security advisories separately.

Community installers are currently unsigned. Obtain releases only from this
repository and verify their SHA-256 checksums. Code signing is not configured.
