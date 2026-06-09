# ADR 0001: Use Wails for Desktop Shell

## Status
Accepted

## Date
2026-05-27

## Decision Date
Approx. 2026-04-11

## Context
We need a desktop shell for a local-first application with a Go backend and a
React frontend. Key constraints:
- Reuse Go backend logic without a separate runtime
- Keep binary size reasonable
- Support Windows first and Linux second
- Avoid maintaining two app stacks

## Decision
Use Wails as the desktop shell.

## Consequences

Positive:
- Reuses Go backend directly through IPC
- Smaller and lighter than Electron
- Aligns with current Go-centric stack

Negative:
- Smaller ecosystem than Electron
- WebView platform differences may require extra handling

## Alternatives Considered

### Electron
Pros:
- Large ecosystem and mature tooling

Cons:
- Heavy runtime and larger binaries
- Requires a separate JS app shell

### Tauri
Pros:
- Small binaries
- Strong security posture

Cons:
- Requires Rust in the core toolchain
- Less aligned with the Go backend

### Qt
Pros:
- Native UI toolkit

Cons:
- Higher complexity and heavier UI development

## Notes
- This ADR is retrospective; reasoning is reconstructed from plans and current context.
