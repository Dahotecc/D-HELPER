---
name: post-pr
description: Build or update the single pull request dev -> main of a D-* repo with a complete French description. Use it ONLY when the user asks for the pull request. Commits and pushes on dev are allowed at any time without this skill. Never merge without an explicit request from the user.
---

# post-pr: build or update the dev -> main pull request (on request only)

Use this procedure only when the user asks for the pull request.
The user gave a standing authorization to commit on `dev` and to push `dev` at any time.
The pull request itself (create or edit) is done only on request.
The user did NOT give an authorization to merge. Merge only when the user asks for it in this session.

Conventions: refer to D-OPS `docs/conventions.md` (sections 9, 12, 13, 14).
Write the pull request text in French. Do not use em dashes or en dashes.

## 1. Make sure that the state is correct

1. Make sure that the current branch is `dev`. If it is not `dev`, stop and tell the user.
2. Run `git fetch origin`. If `origin/dev` has commits that are not in local `dev`, stop and tell the user.
3. If there are uncommitted changes, show them to the user. Commit them in atomic commits with the convention `<Domaine> : <description>`.
   - A change that the users can see gets a clear subject without a technical prefix. These subjects become the release notes.
   - A technical change uses `CI`, `Docker`, `Git`, `Doc`, `Santé`, `Sauvegarde`, `Config` or `Style`.
   - End each commit message with the `Co-Authored-By` trailer of the session.
4. Make sure that no secret is in the commits of `origin/main..dev` (tokens, `.env`, `d-hub.config.json`, `pb_data`). If you find a secret, stop and tell the user.

## 2. Do the local checks

Run the checks in the Go container (Go is not installed on the computer):

```bash
docker run --rm -u "$(id -u):$(id -g)" -e HOME=/tmp -e CGO_ENABLED=0 -v "$PWD":/src -w /src golang:1.27.1-alpine3.24 sh -ec '
  test -z "$(gofmt -l .)"
  go vet ./...
  GOOS=windows GOARCH=amd64 go vet ./...
  go test ./...
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -H=windowsgui -X main.version=dev" -o dist/d-helper.exe .
'
```

- D-HELPER: `gofmt`, `go vet` (Linux and Windows), `go test` and the Windows build must pass.

If a check fails, stop. Do not push. Tell the user what failed.

## 3. Push dev

```bash
git push origin dev
```

## 4. Collect the data for the description

```bash
# All the commits that go to production.
git log --no-merges --format='%h %s' origin/main..dev
# Release notes that the users will see (same filter as the deploy workflow).
git log --no-merges --format='%s' origin/main..dev \
  | grep -Ev '^(CI|Docker|Git|Doc|Santé|Sauvegarde|Config|Style) : ' || true
# Changes that need a manual action before or after the merge.
git diff origin/main..dev -- .env.example docker-compose.yml Dockerfile
```

From the diff, list each stack variable that was added, removed or renamed. The user must set these variables in the Portainer stack BEFORE the merge.

## 5. Create or edit the pull request

Find the open pull request:

```bash
gh pr list --base main --head dev --state open --json number,body,url
```

- If there is no open pull request: create it with `gh pr create --base main --head dev --title "<title>" --body-file <file>`.
- If there is an open pull request: write the full body again from the current data with `gh pr edit <number> --title "<title>" --body-file <file>`. Keep the old entries of the "Journal des sessions" section, and add the entry of this session at the top.

Title: `Mise en production : <short summary of the main changes>`.

Body template (French):

```markdown
## Résumé
<2 to 5 sentences: what changes for the users and for the server>

## Ce que verront les utilisateurs
<release notes list, exactly as computed in step 4; "Aucune note visible" if the list is empty>

## Changements techniques
<grouped list: Docker, CI, Config, code, documentation>

## Actions manuelles avant le merge
<Portainer variables to add or change, data migration, other; "Aucune" if none>

## Vérifications
- [x] <local checks of step 2, with the result>
- [ ] CI verte sur la PR
- [ ] Relecture du diff par l'utilisateur

## Après le merge
- Suivre le workflow deploy (build, push GHCR, webhook Portainer, /healthz).
- Rollback : `IMAGE_TAG=sha-<previous version>` dans la stack Portainer, puis "Update the stack".

## Journal des sessions
### <AAAA-MM-JJ> : <one-line summary of this session>
- <commits of this session>
<older entries, unchanged>

## Commits
<full list of step 4: hash and subject>
```

## 6. Wait for the CI and give the report

1. Run `gh pr checks <number> --watch`.
2. Give the user, in French:
   - the URL of the pull request;
   - the CI result;
   - the release notes that the users will see;
   - the manual actions before the merge.
3. Ask: "Je merge ?" Do not merge before a clear yes.

## 7. Merge (only after a clear yes from the user)

```bash
gh pr merge <number> --merge
```

- Use a merge commit. Do not use squash or rebase: the release notes need the commit subjects.
- Then follow the deploy workflow with `gh run watch` on the run of `main`, and give the result.
- After the merge, run `git fetch origin` and make sure that `dev` and `main` are in sync.
