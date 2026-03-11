---
name: release
description: Cut a new release of the terraform-provider-s2 by tagging and pushing. Triggers the GitHub Actions goreleaser workflow.
user-invocable: true
---

You are cutting a release of the terraform-provider-s2. Follow these steps exactly.

## 1. Preflight checks

Run all of these and surface any issues before proceeding:

- Confirm the working tree is clean (`git status`)
- Confirm we are on `main` and up to date with origin (`git fetch origin main && git status`)
- Confirm `go build ./...` passes
- Show the last 5 tags (`git tag --sort=-v:refname | head -5`) and the commits since the last tag (`git log <last-tag>..HEAD --oneline`)

## 2. Determine the next version

If the user provided a version (e.g. `/release v0.2.0`), use that. Otherwise:
- Look at commits since the last tag
- Suggest a version following semver: patch bump for fixes/chores, minor bump for new features, major bump for breaking changes
- Ask the user to confirm before proceeding

## 3. Create and push the tag

```bash
git tag <version>
git push origin <version>
```

## 4. Confirm

- Show the tag that was pushed
- Remind the user that the GitHub Actions release workflow will now run and handle the goreleaser build and publish
- Provide the Actions URL: https://github.com/s2-streamstore/terraform-provider-s2/actions/workflows/release.yml
