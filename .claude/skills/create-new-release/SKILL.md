---
name: create-new-release
description: Create and push a new semver release tag for pg_watcher. Asks for release type (patch/minor/major), validates commit exists on remote, bumps version, and pushes the tag.
---

# Create New Release

Create and push a new semantic version release tag for pg_watcher.

## Steps

1. **Ask release type**: Use AskUserQuestion to ask if this is a patch, minor, or major release.

2. **Ensure we are on main and up to date**:
   ```bash
   git checkout main
   git pull origin main
   ```

3. **Fetch latest tags from remote**:
   ```bash
   git fetch --tags origin
   ```

4. **Get the latest semver tag** (assumes `v` prefix like `v1.2.3`):
   ```bash
   git tag --sort=-version:refname -l 'v*' | head -1
   ```

5. **Parse and bump version** based on user selection:
   - **patch**: `v1.2.3` → `v1.2.4`
   - **minor**: `v1.2.3` → `v1.3.0`
   - **major**: `v1.2.3` → `v2.0.0`

6. **Validate current commit exists on remote**:
   ```bash
   git fetch origin
   git branch -r --contains HEAD
   ```
   If no remote branch contains HEAD, abort with an error asking the user to push their commits first.

7. **Run tests before releasing**:
   ```bash
   make test
   ```
   If tests fail, abort and report the failure. Do not create the tag.

8. **Create and push the tag**:
   ```bash
   git tag -a <new_version> -m "Release <new_version>"
   git push origin <new_version>
   ```

9. **Report success** with the new tag name.

## Version Parsing

Parse version from tag like `v1.2.3`:
- Extract numbers after `v` prefix
- Split by `.` to get major, minor, patch components
- Handle edge case: if no tags exist, start at `v0.1.0`

## Project-Specific Notes

- **VCS**: GitHub (`git@github.com:maratos-ORG/pg_watcher.git`)
- **Main branch**: `main`
- **Tag format**: `v<major>.<minor>.<patch>` (e.g. `v1.0.3`)
- **CI**: GitHub Actions runs tests on PRs (`pr-tests.yml`); tag push does not trigger any automated pipeline
- **Binary version**: the built binary embeds the git tag via `./bin/pg_watcher -version`
- **Test commands**:
  - `make test` — unit tests with race detection and coverage
  - `make test_pg_watcher` — integration test (requires Docker)
  - `make test_telegraf` — full stack test (requires Docker)
  - `make test_all` — all of the above

## Error Handling

- If current commit is not on remote: "Current commit is not pushed to remote. Please push your changes first."
- If tests fail: "Tests failed. Fix the failures before releasing."
- If tag already exists: "Tag already exists. Please check existing tags."
- If git operations fail: Report the specific error.
