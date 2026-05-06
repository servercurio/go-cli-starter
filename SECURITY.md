# Security Policy

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

This repository uses GitHub's [private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability) to receive reports privately. To submit a report:

1. Open <https://github.com/servercurio/go-echo-starter/security/advisories/new>.
2. Provide a clear description of the issue, the affected component, and a minimal reproduction (commit hash, configuration, sample request, etc.).
3. Suggest an impact assessment if you are able (CVSS vector or plain-language severity).

If you cannot use GitHub's reporting flow, email the maintainer at **nathan@theklickfamily.com** with the subject line `[SECURITY] go-echo-starter`. Encrypted mail is welcome; ask for a current public key in your initial (unencrypted) message.

## What to Expect

- **Acknowledgement** within 3 business days of receipt.
- **Initial triage** (severity assessment, scope confirmation) within 7 business days.
- **Status updates** at least every 14 days while the report is open.
- **Coordinated disclosure** through a GitHub Security Advisory once a fix is available; reporters are credited unless they request otherwise.

We aim to publish fixes for confirmed vulnerabilities within 90 days of triage. More severe issues are prioritised; complex issues affecting external dependencies may take longer and will be communicated as such.

## Scope

This policy covers the source code in this repository and any release artifact published from its `100-user-deploy-release-artifact.yaml` workflow.

Out of scope:

- Vulnerabilities in third-party dependencies. Report those upstream; if exploitation requires this project to expose them in a non-default way, that part of the chain is in scope.
- Findings from automated scanners without a working proof of concept.
- Configuration choices made by downstream consumers (this is a starter template — production decisions about TLS modes, reverse-proxy trust, auth, etc. are the deploying party's responsibility).

## Supported Versions

Only the latest released minor version receives security fixes. Older versions may be patched at the maintainer's discretion when the fix is trivial to backport.

## Safe Harbor

Good-faith security research conducted under this policy will not result in legal action from the maintainer. "Good faith" means: avoiding privacy violations, data destruction, service disruption, or access beyond what is necessary to demonstrate the issue, and giving us reasonable time to respond before public disclosure.
