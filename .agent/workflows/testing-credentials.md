---
description: Instructions for using specific credentials during browser testing.
---

# Browser Testing Credentials

When testing the KinCart application in a browser, ALWAYS use the official test credentials defined in the `Makefile`. Do NOT use random or guest credentials.

## Credentials

| Role | Username | Password |
| :--- | :--- | :--- |
| **Manager** | `manager_test` | `pass1234` |
| **Shopper** | `shopper_test` | `pass1234` |

## Usage Instructions

1.  **Ensure Data Exists**: Before testing, ensure the test data is seeded by running:
    ```bash
    make seed-test-data
    ```
2.  **Login**: Use the credentials above on the `/login` page.
3.  **Persistence**: These credentials should be used for all automated and manual browser interactions performed by the agent.

// turbo-all
## Setup Command
1. Run `make seed-test-data` to ensure credentials are valid in the local database.
