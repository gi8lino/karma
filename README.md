# karma - Kustomization Resource Management Assistant

karma keeps nested `kustomization.yaml` files in sync with the directory tree while preserving comments, respecting `.gitignore`, and honoring user-provided skip rules.

## Usage

```sh
karma [options] <base-dir>...
```

## Flags

- `-s`, `--skip` – Accepts comma-separated patterns; supports `*` wildcards, `/*` to skip a directory’s kustomization without entering it, and `/**` to skip the kustomization but still descend into its children (so those nested dirs can still be handled separately).
- `-v` – Increase verbosity to show resource diffs.
- `-vv` – Enable verbose mode so `[NO-OP]` and `[SKIPPING]` appear.
- `--mute`, `-q` – Silence all logging (summary, diffs, and status lines); this flag conflicts with `-v`/`-vv`.
- `--order` – Customize the ordering of remote, directory, and file groups (default `remote,dirs,files`).
- `--no-gitignore`, `-g` – Disable per-directory `.gitignore` evaluation.
- `--include-dot`, `-i` – Include dotfiles and dot-directories.
- `--no-dir-slash`, `-D` – Keep directory resources without a trailing slash.

## Logging

- Default output shows `[PROCESS]`, `[UPDATED]`, and `[SUMMARY]`.
- `-v` adds the resource diff (`-  - foo` / `+  - bar` lines).
- `-vv` ups the level so `[NO-OP]` and `[SKIPPING]` appear as well.
- `--mute`, `-q` shuts logging off entirely.

## Features

- Writes only the `resources` block, preserving other fields and comments.
- Supports remote resources, optional directory suffixing, alphabetical ordering, and fast `skip` patterns.
- Reads `.gitignore` files from each directory figure to allow fine-grained exclusions.
- Plans and updates per base directory, reporting a final summary.

## Testing

```sh
GOPROXY=off GOSUMDB=off GOCACHE=/tmp/go-build go test ./...
```

## Releases

- Builds use [goreleaser](https://goreleaser.com/) targeting macOS (amd64/arm64) and Windows (amd64/arm64) binaries.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
