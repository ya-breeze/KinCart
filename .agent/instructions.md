# Agent Instructions for KinCart

This file contains project-specific rules for the AI agent (Antigravity).

## 1. Knowledge Base
Consult the topical memory in `.agent/memory/` for deep context on specific areas:
- **[Architecture & Isolation](file:///Users/ek/work/KinCart/.agent/memory/architecture.md)**: Data scoping, background workers, and SQL rules.
- **[Business Logic](file:///Users/ek/work/KinCart/.agent/memory/logic.md)**: Flyer linking, item protection, and parsing service patterns.
- **[Frontend & PWA](file:///Users/ek/work/KinCart/.agent/memory/frontend.md)**: UX standards, lazy loading, and Share Target implementation.
- **[Testing Strategy](file:///Users/ek/work/KinCart/.agent/memory/testing.md)**: Playwright patterns, IDB injection, and final validation rules.

## 2. Browser Testing & Authentication
- **NEVER** use random or guest credentials for testing.
- **ALWAYS** use the official test credentials found in [testing-credentials.md](file:///Users/ek/work/KinCart/.agent/workflows/testing-credentials.md).
- **Default Credentials**:
  - Manager: `manager_test` / `pass1234`
  - Shopper: `shopper_test` / `pass1234`

## 3. Environment Setup
- Before running browser tests, ensure the backend is running and data is seeded.
- Use `make seed-test-data` to reset/seed test users.

## 4. Workflows
- Refer to `.agent/workflows/` for automated tasks like quality checks (`check-quality.md`) or system learning (`learn.md`).
