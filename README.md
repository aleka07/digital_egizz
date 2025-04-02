# Digital Egiz

A user-friendly, self-hostable framework for creating and managing Digital Twins. Digital Egiz utilizes Eclipse Ditto as its core Digital Twin platform, provides a Go-based backend for API interactions and logic, and aims to add a modern JavaScript frontend in future versions.

## Overview

Digital Egiz simplifies the creation, management, and interaction with digital twins by providing:

- A complete, containerized Eclipse Ditto setup for digital twin platform functionality
- A Go-based REST API for simplified interaction with digital twins
- Docker Compose orchestration for easy deployment on any system

## Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for development only)

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/digital-egiz/digital-egiz.git
   cd digital-egiz
   ```

2. Start the services:
   ```bash
   docker-compose up -d
   ```

3. Access the services:
   - Go Backend API: http://localhost:9090
   - Eclipse Ditto API: http://localhost:8080
   - Swagger UI: http://localhost:8081

## API Endpoints

The Digital Egiz API provides the following endpoints:

### Health Check
- `GET /health` - Check if the service is running properly

### Digital Twin Operations
- `POST /api/v1/twins` - Create a new digital twin
- `GET /api/v1/twins/{id}` - Retrieve a digital twin by ID
- `PUT /api/v1/twins/{id}` - Update a digital twin
- `DELETE /api/v1/twins/{id}` - Delete a digital twin
- `GET /api/v1/twins` - List all digital twins with pagination

## Authentication

Digital Egiz uses the authentication mechanism provided by Eclipse Ditto. The default credentials are:

- Username: `ditto`
- Password: `ditto`

These credentials can be changed in the `docker-compose.yml` file.

## Development

### Backend Development

To run the Go backend locally:

```bash
cd backend
go mod tidy
go run cmd/main.go
```

### Building Docker Images

To build the Docker images:

```bash
docker-compose build
```

## Project Structure

```
.
├── backend/                 # Go backend service
│   ├── api/                 # API definitions
│   ├── cmd/                 # Entry points
│   ├── internal/            # Internal packages
│   ├── pkg/                 # Shared packages
│   └── Dockerfile           # Backend Docker configuration
├── docker/                  # Docker configurations
│   └── ditto/               # Eclipse Ditto configuration
├── docker-compose.yml       # Main Docker Compose file
└── README.md                # Project documentation
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. 