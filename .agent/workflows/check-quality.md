---
description: Ensure code quality by running tests and linting
---

After making changes to the codebase, always run the following commands to ensure correctness and code quality:

For backend changes:
```bash
make lint-backend
make test-backend
```

For frontend changes:
```bash
make lint-frontend
make test-frontend
```

For full project verification:
```bash
make lint
make test
```
