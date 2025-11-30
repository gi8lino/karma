# kustomizer

kustomizer is a small tool to keep nested `kustomization.yaml` files in sync with the directory layout while respecting `.gitignore` and custom skip patterns.

## Usage

```sh
go run cmd/kustomizer/main.go [options] <base-dir>...
```

### Flags

- `-s`, `--skip` – Accepts comma-separated patterns (supports `*` wildcards and `/**`/`/*` suffixes).
- `-v` – Increase verbosity (use `-vv` to enable tracing output).
- `--no-gitignore` – Disable `.gitignore` handling.
- `--include-dot` – Include dotfiles and directories in the scan.
- `--no-dir-slash` – Keep directory entries without a trailing slash.
- `--no-dir-first` – Disable directory-first ordering.

## Features

- Preserves existing YAML comments and order while touching only the `resources` field.
- Keeps remote resources intact and sorts directories/files per configuration.
- Honors `.gitignore` files (per directory) unless disabled.
- Offers skip patterns for files, directories, and entire subtrees.

## Testing

Run `go test ./...` to exercise the CLI, processor, gitignore, and logging helpers.
