# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`binary-download-service` is a FastAPI service for uploading, storing, and distributing binary files (primarily node_exporter and node-push-exporter binaries for various OS/architecture combinations).

## Commands

```bash
# Install dependencies
pip install -r requirements.txt

# Run the service
./run.sh
# or directly:
uvicorn main:app --host 0.0.0.0 --port 8080

# Run with auto-reload for development
uvicorn main:app --reload --host 0.0.0.0 --port 8080
```

## Architecture

The service follows a simple layered architecture:

- **main.py**: FastAPI application with REST endpoints, file parsing logic, and static file mounting
- **database.py**: SQLAlchemy ORM model (`FileRecord`) and database session management using SQLite
- **schemas.py**: Pydantic response models for API validation
- **static/index.html**: Single-page web UI for file management

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/files` | List all files, optionally filtered by `program` query param |
| POST | `/api/upload` | Upload a binary file (multipart form) |
| DELETE | `/api/files/{file_id}` | Delete a file by ID |
| GET | `/download/{filename}` | Download a file |

### Filename Parsing

The service parses filenames to extract metadata using this pattern:
```
{program}-{version}-{os}-{arch}[.tar.gz]
```

Supported programs: `node_exporter`, `node-push-exporter`
Supported OS: `linux`, `darwin`
Supported architectures: `amd64`, `arm64`

### Data Storage

- **Database**: SQLite (`files.db`) - stores file metadata
- **File storage**: Local filesystem (`uploads/` directory)
- Both are created relative to the project root

## Key Behaviors

- Files are stored in `uploads/` with their original filenames
- Upload replaces existing files with the same filename (updates DB record)
- CORS is enabled for all origins (development-friendly)
- The web UI serves as a SPA at the root path, with API routes under `/api/`