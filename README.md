# karma - Kustomization Resource Management Assistant

karma keeps nested `kustomization.yaml` files in sync with the directory tree while preserving comments, respecting `.gitignore`, and honoring user-provided skip rules.

## Usage

```sh
karma [options] <base-dir>...
```

## Flags

- `-s`, `--skip` – Completely ignore matching resources. Accepts comma-separated patterns and wildcards.
- `--opaque` – Keep matching directories as resources but do not descend into them.
- `--preserve` – Keep matching directories as resources and descend into them, but do not rewrite their own kustomization.
- `-v` – Increase verbosity to show resource diffs.
- `-vv` – Enable verbose mode so `[NO-OP]` and `[SKIPPING]` appear.
- `--mute`, `-q` – Silence all logging (summary, diffs, and status lines); this flag conflicts with `-v`/`-vv`.
- `--order` – Customize the ordering of external, directory, and file groups (default `external,dirs,files`).
- `--no-gitignore`, `-g` – Disable `.gitignore` processing.
- `--include-dot`, `-i` – Include dotfiles and dot-directories.
- `--suffix`, `-x` – Append `/` when listing directories.
- `--prefix`, `-p` – Prefix directory entries with `./`.
- `--dry-run` – Show the changes Karma would make without writing files.
- `--check` – Do not write files and exit non-zero when any kustomization is out of sync.

## Logging

- Default output shows `[PROCESS]`, `[UPDATED]`, and `[SUMMARY]`.
- `-v` adds the resource diff (`-  - foo` / `+  - bar` lines).
- `-vv` ups the level so `[NO-OP]` and `[SKIPPING]` appear as well.
- `--mute`, `-q` shuts logging off entirely.
- ANSI colors are used only for terminal output; redirected output is plain text and `NO_COLOR` is respected.

## Features

- Writes only the `resources` block, preserving other fields and comments.
- Preserves external resource references such as remote URLs and non-direct local paths, supports optional directory suffixing, alphabetical ordering, and explicit `skip`, `opaque`, and `preserve` patterns.
- Reads `.gitignore` files from each directory figure to allow fine-grained exclusions.
- Plans and updates per base directory, reporting a final summary.
- Supports non-mutating `--dry-run` previews and CI-friendly `--check` validation.

## Testing

```sh
make fmt        # Format source files.
make fmt-check  # Verify formatting without modifying files.
make test       # Run fmt-check, vet, and unit tests.
```

## Releases

- Builds use [goreleaser](https://goreleaser.com/) targeting Linux, macOS, and Windows on amd64 and arm64.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
