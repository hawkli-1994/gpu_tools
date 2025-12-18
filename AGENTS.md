# Repository Guidelines

## Project Structure & Module Organization
Source lives under `pkg/gpu`, organized per vendor (`nvidia`, `amd`, `ix`, `mx`, `enflame`, `cpu`, etc.) plus shared types in `pkg/gpu/types.go` and the loader registry in `pkg/gpu/registry.go`. Each vendor package usually keeps sample CLI outputs in `testdata/` (for example `pkg/gpu/nvidia/testdata/nvidia.txt`) to keep fixtures close to the implementation.

## Build, Test, and Development Commands
- `go test ./...` &mdash; runs the entire suite, including integration-style loader tests that parse the fixtures under `pkg/gpu/*/testdata`.
- `go build ./...` &mdash; ensures every package (including vendor-specific loaders) compiles before you post a change.
- `go vet ./...` &mdash; lightweight linting that catches common mistakes; run after major refactors.
- `gofmt -w pkg` &mdash; formats Go sources; apply before committing to keep diffs clean.

## Coding Style & Naming Conventions
Follow idiomatic Go: tabs for indentation, camel-cased exported identifiers (e.g., `GPUInfoLoader`), and short, descriptive locals. New loaders should implement `Available`, `Load`, and `Vendor` just like `pkg/gpu/nvidia/nvidia.go`. Keep error messages actionable (reference the vendor command that failed) and prefer early returns over deeply nested branches. Place helper structs in the same file unless they are shared across packages, in which case move them into `pkg/gpu/types.go`.

## Testing Guidelines
Every loader needs `_test.go` files mirroring the implementation file name (`nvidia_test.go`, `amd_test.go`, etc.). Tests should load fixture files from the co-located `testdata` directory so they run offline; see `pkg/gpu/enflame/enflame_test.go` for an example pattern. Add new fixtures whenever CLI output changes and keep filenames descriptive (`efs-15.txt`, `mem_info_vram_total`). Maintain full `go test ./...` parity before pushing; aim for coverage that touches parsing and registry registration paths.

## Commit & Pull Request Guidelines
Commits are short, present-tense summaries (`fix ix gpu import error`, `Update go.yml`). Use ≤72 character subjects, optionally prepend the touched vendor (`nvidia:`, `amd:`) when the change is scoped. In pull requests, include: purpose summary, notable implementation points (e.g., “adds mx loader parsing for `mx-smi`”), reproducible test output (`go test ./...`), and links to related issues. Screenshots aren’t required unless documenting CLI output differences. Make sure branch diffs are linted/formatted before requesting review.

## Agent Workflow Tips
Vendor loaders shell out to hardware tools (`nvidia-smi`, `rocm-smi`, `efsmi`, etc.). When developing locally without GPUs, rely on the fixtures and avoid committing machine-specific paths. If you add a new vendor, initialize its package under `pkg/gpu/<vendor>` and register it in `init()` so `gpu.GetAllGPULoaders()` picks it up automatically.
