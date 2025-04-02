# Project Configuration (LTM) - Digital Egiz

This file contains the stable, long-term context for the project. It should be updated infrequently, primarily when core goals, tech, or patterns change.

## Core Goal

Create a user-friendly, self-hostable framework named "Digital Egiz" for creating and managing Digital Twins. The framework utilizes Eclipse Ditto as its core Digital Twin platform, provides a Go-based backend for API interactions and logic, and features a modern JavaScript frontend with potential for future 3D visualization and AI integrations. The entire system must be easily deployable via Docker Compose. The initial focus is on delivering a stable MVP (Minimum Viable Product).

## Tech Stack

*   **Digital Twin Platform:** Eclipse Ditto (running via Docker Compose)
*   **Backend:** Go (Golang) - using standard library for HTTP, potentially `gorilla/mux` or `chi` for routing later.
*   **Frontend:** JavaScript (Specific framework like React, Vue, or Svelte TBD). Focus on modern UI/UX.
*   **Orchestration:** Docker, Docker Compose
*   **Database (for Ditto):** MongoDB (managed within Ditto's Docker Compose setup)
*   **Linting/Formatting (Backend - Go):** Standard `gofmt`, `go vet`, potentially `golangci-lint`.
*   **Linting/Formatting (Frontend - JS):** ESLint, Prettier (to be configured when frontend is added).
*   **Testing (Backend - Go):** Standard Go testing library.
*   **Testing (Frontend - JS):** TBD (e.g., Jest, Vitest).

## Critical Patterns & Conventions

*   **API Interaction with Ditto:** All interactions with the Eclipse Ditto API should occur through the Go backend service. The backend acts as a proxy and business logic layer.
*   **Dockerization:** All core components (Ditto, Go backend, Frontend) MUST run within Docker containers orchestrated by a single `docker-compose.yml` file in the project root.
*   **Configuration:** Backend configuration (like Ditto API endpoints, credentials) should be primarily managed via environment variables, injected through `docker-compose.yml`.
*   **API Design (Backend):** Aim for clear, RESTful principles for the API exposed by the Go backend to the frontend.
*   **Error Handling (Backend):** Implement consistent error handling in the Go backend.
*   **Commit Messages:** Follow Conventional Commits format (e.g., `feat:`, `fix:`, `chore:`, `docs:`).
*   **Development Approach:** Iterative development following MVP principles. Small, testable steps with frequent commits.

## Key Constraints

*   **Self-Hosted Focus:** The primary deployment target is a user running `docker-compose up` on their own machine/server. Not designed as a multi-tenant SaaS initially.
*   **Ditto Dependency:** The system fundamentally relies on Eclipse Ditto.
*   **MVP First:** Avoid premature implementation of advanced features (3D, complex AI) until the core MVP is stable and functional.
*   **Go Backend:** Backend logic must be implemented in Go.
*   **JS Frontend:** Frontend must be implemented in JavaScript/TypeScript.