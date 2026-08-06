# Teambition Skill

Codex skill for Teambition project, task, and sprint operations. It uses the
official UserToken CLI for ordinary operations and includes a small Go client
for AppToken-only sprint APIs.

## Layout

- `SKILL.md`: operating instructions and safety rules
- `scripts/teambition_app`: launcher for the AppToken client
- `scripts/teambition_app.go`: direct OpenAPI client
- `references/`: TQL references

## Install for Codex

Keep this repository in a normal workspace and expose it through Codex's skill
directory:

```sh
ln -s /Users/yang/work/teambition-skill /Users/yang/.codex/skills/teambition
```

## Test

Requires Go 1.22 or later:

```sh
ASDF_GOLANG_VERSION=1.22.12 go test \
  -ldflags=-linkmode=external \
  scripts/teambition_app.go scripts/teambition_app_test.go
```

## Credentials

Do not commit UserToken, App Secret, JWT, or local environment files. Configure
the required `TEAMBITION_*` variables outside this repository; see `SKILL.md`
for the exact requirements.
