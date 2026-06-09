# Authentication

## Purpose
Authenticate family members and manage their sessions so only authorized users can access the app.

## Requirements

### Requirement: Login

A user SHALL authenticate with a username and password before accessing the app.

#### Scenario: Successful login
- **GIVEN** a user exists with a valid username and password
- **WHEN** the user submits the login form
- **THEN** they are redirected to the Dashboard

#### Scenario: Failed login — wrong password
- **GIVEN** a user exists
- **WHEN** the user submits the login form with an incorrect password
- **THEN** an error message is displayed and the user stays on the login page

#### Scenario: Failed login — unknown username
- **WHEN** the user submits the login form with a username that does not exist
- **THEN** an error message is displayed and the user stays on the login page

#### Scenario: Rate limiting
- **WHEN** a login attempt is made more than 5 times per minute from the same IP
- **THEN** the server returns 429 Too Many Requests

#### Scenario: Protected page redirects unauthenticated user
- **GIVEN** the user is not logged in
- **WHEN** the user navigates directly to a protected URL (e.g., `/lists/:id`)
- **THEN** they are redirected to the login page

---

### Requirement: Session persistence

Sessions SHALL survive page reloads without requiring re-login.

#### Scenario: Page reload keeps session
- **GIVEN** the user is logged in
- **WHEN** the page is reloaded
- **THEN** the user remains logged in and on the current page (or Dashboard)

#### Scenario: Access token auto-refresh
- **GIVEN** the user's access token has expired (15-minute TTL)
- **WHEN** the user makes an API request
- **THEN** the token is refreshed automatically and the request succeeds

---

### Requirement: Logout

Logging out SHALL clear the session and prevent further access.

#### Scenario: Successful logout
- **GIVEN** the user is logged in
- **WHEN** the user clicks Logout
- **THEN** they are redirected to the login page and cannot access protected pages

#### Scenario: Refresh token reuse detection
- **GIVEN** a refresh token has already been used once
- **WHEN** the same refresh token is used again
- **THEN** all sessions for that user are revoked and the user must log in again
