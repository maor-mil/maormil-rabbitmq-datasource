---
name: release-plugin
description: Prepare and cut a release of the signed RabbitMQ data source — choose the version, update CHANGELOG and package.json, validate, tag, and follow the release workflow up to submitting the new version to the Grafana plugin catalog. Use when asked to release, publish or bump the plugin version.
---

# Release the plugin

The plugin version comes from `package.json` (`%VERSION%` in `src/plugin.json` is replaced at build time).
Releases are built and signed by `.github/workflows/release.yml` (`grafana/plugin-actions/build-plugin`) when a
`v*` tag is pushed; it fails if the tag is not exactly `v` + the `package.json` version.

Pushing a tag and publishing a release are outward-facing: confirm the version and the go-ahead with the user first.

## Steps

1. Make sure `main` is green in CI and contains everything to release.
2. Pick the version (semver): a raised minimum Grafana version, removed settings or changed defaults → major;
   new features → minor; fixes → patch.
3. Update `CHANGELOG.md`: rename `## Unreleased` to `## <version>` (keep the Breaking changes / Added / Changed /
   Fixed sections), and add a fresh empty `## Unreleased` on top if desired.
4. Bump the version without creating a tag yet: `npm version <version> --no-git-tag-version`.
5. Validate the build like the catalog will (see the `validate-plugin` skill) and fix any errors.
6. Commit (`chore(release): v<version>`), open a PR, and merge it once CI is green.
7. Tag the merge commit and push the tag (only after the user confirms):
   ```bash
   git checkout main && git pull
   git tag v<version> && git push origin v<version>
   ```
8. Watch the workflow: `gh run watch $(gh run list --workflow release.yml --limit 1 --json databaseId -q '.[0].databaseId')`.
   It signs the plugin with the `POLICY_TOKEN` repository secret (a Grafana Cloud access policy token with the
   `plugins:write` scope) and creates a **draft** GitHub release with the zip and its sha1.
9. Review and publish the draft release, then submit the new version in Grafana Cloud
   (My Account → Plugins → Submit update) with the release zip URL and sha1.
   Docs: https://grafana.com/developers/plugin-tools/publish-a-plugin/publish-or-update-a-plugin.md
