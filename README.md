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

Clone this repository and expose it through Codex's user skill directory:

```sh
mkdir -p "$HOME/work" "$HOME/.agents/skills"
git clone https://github.com/BackFire/teambition-skill.git "$HOME/work/teambition-skill"
ln -s "$HOME/work/teambition-skill" "$HOME/.agents/skills/teambition"
```

Install the official `teambition_cli` and configure credentials as described
in `SKILL.md`. Start a new Codex session to discover the skill.

## Test

Requires Go 1.22 or later:

```sh
ASDF_GOLANG_VERSION=1.22.12 go test \
  -ldflags=-linkmode=external \
  scripts/teambition_app.go scripts/teambition_app_test.go
```

## Credentials

Do not commit UserToken, App Secret, JWT, or local environment files. Configure
the required `TEAMBITION_*` variables outside this repository. The AppToken
launcher automatically loads `teambition.env` and `teambition-app.env` from
`${XDG_CONFIG_HOME:-$HOME/.config}/cli-credentials`; explicitly set non-empty
environment variables take precedence. See `SKILL.md` for the exact
requirements.

## License

MIT. See [LICENSE](LICENSE).
