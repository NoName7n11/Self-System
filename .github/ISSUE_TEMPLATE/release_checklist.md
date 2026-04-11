---
name: Release Checklist
about: Track readiness before cutting a semantic version release.
title: "release: vX.Y.Z"
labels: ["release", "chore"]
---

## Release Metadata

- Target version: vX.Y.Z
- Planned date:
- Release owner:

## Checklist

- [ ] CHANGELOG entry exists for this version
- [ ] Full tests pass locally: go test ./...
- [ ] CI is green on default branch
- [ ] Release workflow exists and is valid
- [ ] Tag is prepared: git tag vX.Y.Z
- [ ] Tag pushed: git push origin vX.Y.Z
- [ ] GitHub Release artifacts generated (Linux/Windows + checksums)
- [ ] Post-release notes reviewed

## Notes

Add rollout notes, known issues, and rollback plan if needed.
