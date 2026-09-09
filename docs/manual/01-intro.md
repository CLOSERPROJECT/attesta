# Introduction

## Vision And Objectives

Attesta is a workflow-driven traceability platform for multi-organization industrial processes.

Its main objective is to let different organizations contribute evidence to the same structured process while keeping:

- clear role-based responsibility
- ordered execution across steps
- stored evidence and attachments
- notarized outputs for each completed substep
- a Digital Product Passport, when configured

In practice, Attesta is designed to answer a simple question: who submitted what, in which order, under which organizational responsibility, and what traceability package can be exported at the end.

## Architecture

Attesta is a small end-to-end application composed of the following parts.

### Backend

- A Go server built on `net/http`
- HTML rendered with Go templates
- Route handlers for login, stream selection, process pages, admin consoles, downloads, and DPP pages

### Frontend

- A Vite-built JavaScript and CSS bundle
- HTMX for partial refreshes in server-rendered pages
- SSE for live updates when stream instances change
- An embedded stream composer UI based on Formata Arch, exposed inside Attesta as the composer

### Persistence

- MongoDB stores stream instances, substep progress, notarization records, and saved stream definitions
- Mongo GridFS stores uploaded files and attachments

### Identity And Organization Layer

- Appwrite is used for authentication, sessions, membership handling, invites, password recovery, and organization-linked user state
- Organizations, memberships, and role catalogs are exposed in Attesta through admin consoles

### Authorization

- Cerbos enforces whether a substep may be completed
- Cerbos also protects the platform admin console, org admin console, stream composer access, and stream deletion

### Live Traceability Output

- Every completed substep produces a notarization record with a SHA-256 digest
- Attesta can export a notarized JSON package
- Attesta can export a Merkle tree built from notarized substep records
- If DPP is enabled, Attesta generates a GS1 Digital Link page for the completed stream instance

## Standards

Attesta currently aligns with two major standards-oriented traceability mechanisms.

### GS1 DPP

Attesta can generate a Digital Product Passport URL using GS1 Digital Link format:

`/01/{gtin}/10/{lot}/21/{serial}`

The DPP payload is generated from stream configuration plus values collected during the stream. In typical setups:

- `gtin` comes from stream configuration
- `lot` comes from a configured input key, or a default value
- `serial` comes from a configured strategy, such as the process ID

The DPP page exposes traceability data, organizations, roles, substep values, and attachment links.

### Merkle Tree For DPP Evidence

Attesta computes a Merkle tree from the notarized substep export. This gives a compact integrity structure over the completed evidence set:

- each notarized substep becomes a Merkle leaf
- leaves are hashed and combined into tree levels
- the tree produces a single root hash

This is useful when you want to prove that the exported evidence package has not changed after generation.

## Actor Model

Attesta works through three main personas:

- [Platform Admin](./03-platform-admin.md)
- [Org Admin](./04-org-admin.md)
- [User](./05-user.md)

The difference between users and roles is important:

- a user is an account
- a role is a responsibility assigned to that account inside an organization

## How Streams Fit In

The operational heart of Attesta is the stream system:

- a stream defines the process
- a stream instance executes the process
- a completed stream instance can produce a DPP and export package

See [Streams](./02-streams.md).

