## ADDED Requirements

### Requirement: Pause request before forwarding in review mode
The system SHALL intercept an incoming HTTP request and pause forwarding when a route has `reviewMode = true`.

#### Scenario: Request sent to reviewer channel
- **WHEN** a request arrives at a route with `reviewMode = true`
- **THEN** the proxy sends the full request (method, URL, headers, body) to an in-memory review channel and pauses the response timer

#### Scenario: Reviewer can approve request for forwarding
- **WHEN** the reviewer signals approval on the review channel
- **THEN** the proxy resumes, forwards the original request to the upstream, and returns the upstream response to the client

#### Scenario: Reviewer can discard request
- **WHEN** the reviewer signals discard on the review channel
- **THEN** the proxy returns HTTP 403 Forbidden to the client without forwarding to upstream

### Requirement: Pause response before returning to client in review mode
The system SHALL intercept the upstream response and pause delivery to the client when `reviewMode = true` is set for response review.

#### Scenario: Response sent to reviewer channel
- **WHEN** the upstream returns a response for a route with `reviewMode = true` (response review enabled)
- **THEN** the proxy sends the response (status code, headers, body) to the review channel and pauses client delivery

#### Scenario: Reviewer can approve response for client delivery
- **WHEN** the reviewer signals approval on the response review channel
- **THEN** the proxy forwards the original upstream response to the client unchanged

### Requirement: Review mode is opt-in per route
The system SHALL not enable review interception by default. Only routes explicitly marked with `reviewMode = true` trigger review behavior.

#### Scenario: Normal routes bypass review entirely
- **WHEN** a request arrives at a route without `reviewMode` enabled
- **THEN** the proxy forwards the request directly to upstream and streams the response back to the client with no interception
