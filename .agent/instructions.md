# Agent Instructions for KinCart

This file contains project-specific rules and instructions for the AI agent (Antigravity).

## 1. Browser Testing & Authentication
- **NEVER** use random or guest credentials for testing.
- **ALWAYS** use the official test credentials found in the `Makefile` or `.agent/workflows/testing-credentials.md`.
- **Default Credentials**:
  - Manager: `manager_test` / `pass1234`
  - Shopper: `shopper_test` / `pass1234`

## 2. Environment Setup
- Before running browser tests, ensure the backend is running and data is seeded.
- Use `make seed-test-data` to reset/seed test users.

## 3. Workflow References
- Refer to `.agent/workflows/` for specific task-oriented instructions.
