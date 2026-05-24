---
name: Bug report
about: Create a report to help us improve
labels: bug
---

## Describe the bug
What broke, and what were you trying to do?

## To Reproduce
Steps to reproduce the behavior:
1.
2.
3.

## Expected behavior
What you expected to happen.

## Actual behavior
What actually happened. Include errors, stack traces, and relevant logs if
possible.

## Diagnostics
If the bundle starts, please run the experimental doctor command and attach the
terminal output plus the generated JSON report:

```sh
docker exec any-sync-bundle any-sync-bundle doctor
```

The report is usually written to `./data/doctor/doctor_<timestamp>.json` on the
host. Please remove secrets before attaching configs or logs.

## Environment
- Deployment: [AIO container | minimal container | binary]
<!-- It helps to attach output from `any-sync-bundle --version`. -->
- Version/tag:
- Compose file or start command:
- OS/Arch:
- Storage: [local Badger | S3/MinIO]

## Configuration
<!--
If applicable, add relevant config snippets without private keys or passwords.
-->

## Additional context
Add any other context about the problem here.
