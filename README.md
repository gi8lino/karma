# kustomizer

kustomizer keeps nested `kustomization.yaml` files in sync with the directory tree while preserving comments, respecting `.gitignore`, and honoring user-provided skip rules.

## Usage

```sh
kustomizer [options] <base-dir>...
```

## Flags

- `-s`, `--skip` – Accepts comma-separated patterns; supports `*` wildcards, `/*` to skip a directory’s kustomization without entering it, and `/**` to skip the kustomization but still descend into its children (so those nested dirs can still be handled separately).
- `-v` – Increase verbosity; pass `-vv` for trace logs.
- `--silent`, `-q` – Suppress per-kustomization no-op logs while still emitting summary output. `-q` conflicts with `-v`/`-vv`; only one of the logging flags may be specified in a single invocation.
- `--no-gitignore`, `-g` – Disable per-directory `.gitignore` evaluation.
- `--include-dot`, `-i` – Include dotfiles and dot-directories.
- `--no-dir-slash`, `-D` – Keep directory resources without a trailing slash.
- `--no-dir-first`, `-F` – Disable directory-first ordering.

## Logging

- Colored output tags like `[UPDATED]`, `[SKIPPING]`, `[TRACE]`, and `[SUMMARY]` reflect the action taken.
- Trace mode (`-vv`) emits detailed traversal info and skip reasons, while `-v` shows skips and summary.
- The logger prints resource diffs before rewriting `kustomization.yaml`, showing removed (`-`) and added (`+`) lines.

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
