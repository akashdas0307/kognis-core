# Security Policy

## Reporting Security Vulnerabilities

**Do not report security vulnerabilities through public GitHub issues.**

Instead, please report them privately:

1. Open a **draft** GitHub Security Advisory
2. Or email the project maintainer directly

You should receive a response within 48 hours. If not, please follow up.

## What to Include

- Type of vulnerability (injection, XSS, privilege escalation, etc.)
- Full path of the affected source file(s)
- Step-by-step instructions to reproduce
- Potential impact
- Any possible mitigations you have identified

## Scope

The following are in scope for security reports:

- The Go core daemon (`core/`)
- The Python Plugin SDK (`sdk/python/`)
- Pipeline template schemas (`schemas/`, `pipelines/`)
- gRPC control plane communication
- NATS event bus message handling
- Plugin sandboxing and permission enforcement

The following are out of scope:

- Denial of service (the framework is intended for local/self-hosted use)
- Vulnerabilities in dependencies (report to upstream)
- Social engineering attacks

## Security Architecture Principles

The Kognis Framework follows these security principles:

- **Principle of least privilege:** Plugins declare capabilities in their manifest and can only access what they declare
- **No direct plugin-to-plugin memory access:** All communication goes through the event bus
- **Sandboxed plugin processes:** Each plugin runs in its own process, supervised by the core
- **Local-first:** Designed for local/self-hosted deployment; network exposure is opt-in
- **Authenticated control plane:** gRPC over Unix socket with file permission enforcement

## Known Security Considerations

- Plugin manifests declare permissions; enforcement is by the core daemon
- Emergency bypass channels bypass normal pipeline controls but have strict gating
- LLM inference (via Ollama) happens locally; cloud LLM access is through the Inference Gateway plugin

## Disclosure Policy

When a vulnerability is reported:

1. We will confirm the vulnerability and determine its scope
2. We will develop a fix and coordinate with the reporter on disclosure timing
3. We will release a patch as soon as possible
4. We will credit the reporter (unless they prefer to remain anonymous)

## Comments on this Policy

If you have suggestions for improving this policy, please open an issue.