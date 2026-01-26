# Quick Start Guide

This demo runs MongoDB + Cerbos with Docker Compose, a Go server, and a Vite-built asset bundle.

## Prerequisites
- Docker + Docker Compose
- Go 1.25+
- Node.js 18+

## Start services
```bash
docker compose up -d
```

## Build frontend assets
```bash
cd web
npm install
npm run build
```

## Run the Go server
```bash
cd ../server
go mod tidy
go run ./cmd/server
```

## Open the demo
- Home: http://localhost:3000
- Backoffice: http://localhost:3000/backoffice
