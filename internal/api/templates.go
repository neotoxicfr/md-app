package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Template represents a built-in markdown template.
type Template struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Content     string `json:"content,omitempty"` // omitted in list view
}

// TemplateSummary is the list-view projection (no content).
type TemplateSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

var builtinTemplates = []Template{
	{
		ID:          "blog-post",
		Name:        "Blog Post",
		Description: "Standard blog post with frontmatter, introduction, sections and conclusion",
		Category:    "writing",
		Content: `---
title: "Your Blog Post Title"
date: ` + "2026-01-01" + `
author: "Your Name"
tags: ["markdown", "blogging"]
draft: true
---

# Your Blog Post Title

> A compelling one-liner that hooks the reader.

## Introduction

Start with context. Why should the reader care about this topic? Set the stage
for what's coming and give them a reason to keep reading.

## The Core Idea

Explain your main argument or insight here. Use concrete examples, data, or
anecdotes to support your point.

### Sub-section

Go deeper. Break complex ideas into digestible sub-sections.

` + "```go" + `
// Code examples make technical posts shine
func main() {
    fmt.Println("Hello, reader!")
}
` + "```" + `

## Practical Takeaways

- **Takeaway 1** — Actionable advice the reader can apply today.
- **Takeaway 2** — A second insight with a concrete next step.
- **Takeaway 3** — One more thing to remember.

## Conclusion

Wrap up your key points. Leave the reader with a clear call-to-action or a
thought-provoking question.

---

*Thanks for reading! If you found this useful, share it with a friend.*
`,
	},
	{
		ID:          "meeting-notes",
		Name:        "Meeting Notes",
		Description: "Structured meeting notes with attendees, agenda and action items",
		Category:    "business",
		Content: `# Meeting Notes — [Topic]

**Date:** 2026-01-01
**Time:** 10:00 – 11:00
**Location:** Conference Room / Video Call
**Facilitator:** [Name]

## Attendees

| Name | Role | Present |
|------|------|---------|
| Alice | Product Manager | ✅ |
| Bob | Tech Lead | ✅ |
| Carol | Designer | ❌ |

## Agenda

1. [ ] Review last week's action items
2. [ ] Sprint progress update
3. [ ] Design review for feature X
4. [ ] Open discussion

## Notes

### 1. Last Week's Action Items

- **Alice** — Completed stakeholder interviews ✅
- **Bob** — API endpoint PR still in review ⏳

### 2. Sprint Progress

- On track for 80% of planned stories
- Blocker: dependency on external API (see below)

### 3. Design Review

Key feedback:
- Simplify the onboarding flow
- Add dark mode support to the dashboard

## Decisions

- [ ] Proceed with simplified onboarding mockup
- [ ] Defer dark mode to next sprint

## Action Items

| Owner | Action | Due |
|-------|--------|-----|
| Alice | Share revised onboarding wireframes | 2026-01-03 |
| Bob | Merge API endpoint PR | 2026-01-02 |
| Carol | Review accessibility checklist | 2026-01-05 |

## Next Meeting

**Date:** 2026-01-08 at 10:00
`,
	},
	{
		ID:          "rfc",
		Name:        "RFC / Technical Proposal",
		Description: "Request for Comments template for technical design proposals",
		Category:    "engineering",
		Content: `# RFC: [Title]

| Field | Value |
|-------|-------|
| **Status** | Draft |
| **Author** | [Your Name] |
| **Created** | 2026-01-01 |
| **Updated** | 2026-01-01 |
| **Reviewers** | @alice, @bob |

## Abstract

One paragraph summary of the proposal. What problem does it solve and what is
the high-level approach?

## Motivation

Why is this change needed? Describe the pain points, user stories, or business
requirements that drive this proposal.

### Current State

How does the system work today? What are the limitations?

### Goals

- **G1:** Primary goal
- **G2:** Secondary goal

### Non-Goals

- This RFC does **not** address [out-of-scope concern].

## Detailed Design

### Architecture

Describe the technical approach. Include diagrams where helpful:

` + "```" + `
┌─────────┐     ┌─────────┐     ┌─────────┐
│  Client  │────▶│   API   │────▶│  Store  │
└─────────┘     └─────────┘     └─────────┘
` + "```" + `

### API Changes

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v2/resource | Create resource |
| GET | /api/v2/resource/{id} | Get resource |

### Data Model

` + "```sql" + `
CREATE TABLE resources (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
` + "```" + `

### Migration Strategy

How do we get from the current state to the new state without downtime?

## Alternatives Considered

### Alternative A: [Name]

Brief description and why it was rejected.

### Alternative B: [Name]

Brief description and why it was rejected.

## Security Considerations

- Authentication / authorization impact
- Data privacy implications
- Input validation requirements

## Rollout Plan

1. **Phase 1** — Feature flag for internal testing
2. **Phase 2** — Beta rollout to 10% of users
3. **Phase 3** — General availability

## Open Questions

- [ ] Question 1?
- [ ] Question 2?

## References

- [Link to related doc](#)
- [Link to prior art](#)
`,
	},
	{
		ID:          "readme",
		Name:        "Project README",
		Description: "Comprehensive project README with badges, install, usage and API docs",
		Category:    "project",
		Content: `# Project Name

[![Build](https://img.shields.io/badge/build-passing-brightgreen)]()
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Go Version](https://img.shields.io/badge/go-1.25-00ADD8)]()

> Short one-line description of what the project does.

## Features

- ✅ Feature one
- ✅ Feature two
- ✅ Feature three
- 🚧 Upcoming feature (in progress)

## Prerequisites

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL 16+ (or SQLite for development)

## Installation

` + "```bash" + `
# Clone the repository
git clone https://github.com/yourorg/project.git
cd project

# Install dependencies
go mod download

# Build
go build -o bin/project ./cmd/server
` + "```" + `

## Quick Start

` + "```bash" + `
# Using Docker Compose
docker compose up -d

# Or run directly
export DATABASE_URL="postgres://localhost:5432/project"
./bin/project
` + "```" + `

The server starts on http://localhost:8080.

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| ` + "`HTTP_ADDR`" + ` | Listen address | ` + "`:8080`" + ` |
| ` + "`DATABASE_URL`" + ` | Database connection string | required |
| ` + "`LOG_LEVEL`" + ` | Log verbosity | ` + "`info`" + ` |

## API Reference

### Create Resource

` + "```http" + `
POST /api/resources
Content-Type: application/json

{
  "name": "example",
  "description": "A new resource"
}
` + "```" + `

### List Resources

` + "```http" + `
GET /api/resources?limit=20&offset=0
` + "```" + `

## Development

` + "```bash" + `
# Run tests
go test ./...

# Run with hot-reload
go run ./cmd/server

# Lint
golangci-lint run
` + "```" + `

## Contributing

1. Fork the repository
2. Create a feature branch: ` + "`git checkout -b feat/my-feature`" + `
3. Commit your changes: ` + "`git commit -m 'feat: add my feature'`" + `
4. Push: ` + "`git push origin feat/my-feature`" + `
5. Open a Pull Request

Please follow [Conventional Commits](https://www.conventionalcommits.org/).

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file.
`,
	},
	{
		ID:          "changelog",
		Name:        "Changelog",
		Description: "Keep-a-Changelog format for tracking project changes",
		Category:    "project",
		Content: `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- New feature description

### Changed

- Updated behavior description

### Fixed

- Bug fix description

## [1.1.0] - 2026-01-15

### Added

- Dark mode support for the dashboard
- Export to PDF functionality
- Webhook notifications on file changes

### Changed

- Improved search performance by 3x
- Updated Go to 1.25

### Deprecated

- Legacy v1 API endpoints (will be removed in 2.0.0)

### Security

- Updated dependencies to patch CVE-XXXX-XXXXX

## [1.0.0] - 2025-12-01

### Added

- Initial release
- Markdown editing with live preview
- File CRUD operations
- Multi-format export via Pandoc
- Redis caching for rendered HTML
- API key authentication
- Docker deployment support

[Unreleased]: https://github.com/yourorg/project/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/yourorg/project/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/yourorg/project/releases/tag/v1.0.0
`,
	},
	{
		ID:          "tutorial",
		Name:        "Tutorial / Guide",
		Description: "Step-by-step tutorial with prerequisites, code blocks and diagrams",
		Category:    "writing",
		Content: `# Tutorial: Building a REST API with Go

> **Level:** Intermediate
> **Time:** ~30 minutes
> **Last updated:** 2026-01-01

## Prerequisites

Before starting, make sure you have:

- [ ] Go 1.25+ installed (` + "`go version`" + `)
- [ ] A code editor (VS Code recommended)
- [ ] Basic understanding of HTTP and JSON
- [ ] ` + "`curl`" + ` or a REST client (Insomnia, Postman)

## What You'll Build

A simple REST API for managing a todo list with:
- Create, read, update, delete operations
- JSON request/response format
- In-memory storage

### Architecture Overview

` + "```mermaid" + `
graph LR
    A[Client] -->|HTTP| B[Router]
    B --> C[Handler]
    C --> D[Storage]
    D --> E[(Memory)]
` + "```" + `

## Step 1: Project Setup

Create a new Go module:

` + "```bash" + `
mkdir todo-api && cd todo-api
go mod init todo-api
` + "```" + `

## Step 2: Define the Data Model

Create ` + "`main.go`" + `:

` + "```go" + `
package main

import (
    "encoding/json"
    "net/http"
    "sync"
)

type Todo struct {
    ID    string ` + "`json:\"id\"`" + `
    Title string ` + "`json:\"title\"`" + `
    Done  bool   ` + "`json:\"done\"`" + `
}

var (
    todos = make(map[string]Todo)
    mu    sync.RWMutex
)
` + "```" + `

## Step 3: Add Handlers

` + "```go" + `
func listTodos(w http.ResponseWriter, r *http.Request) {
    mu.RLock()
    defer mu.RUnlock()

    items := make([]Todo, 0, len(todos))
    for _, t := range todos {
        items = append(items, t)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(items)
}
` + "```" + `

## Step 4: Wire Up the Router

` + "```go" + `
func main() {
    http.HandleFunc("GET /todos", listTodos)
    http.HandleFunc("POST /todos", createTodo)
    http.ListenAndServe(":8080", nil)
}
` + "```" + `

## Step 5: Test It

` + "```bash" + `
go run main.go &
curl -s http://localhost:8080/todos | jq .
` + "```" + `

Expected output:

` + "```json" + `
[]
` + "```" + `

## Summary

| Step | What you did |
|------|-------------|
| 1 | Created Go module |
| 2 | Defined data model |
| 3 | Implemented handlers |
| 4 | Set up routing |
| 5 | Tested the API |

## Next Steps

- Add persistent storage (SQLite or PostgreSQL)
- Add input validation
- Add authentication middleware
- Deploy with Docker

## Troubleshooting

### Port already in use

` + "```bash" + `
lsof -i :8080  # Find the process
kill -9 <PID>  # Kill it
` + "```" + `

### Module errors

` + "```bash" + `
go mod tidy
` + "```" + `

---

*Found an error? [Open an issue](https://github.com/yourorg/tutorials/issues).*
`,
	},
	{
		ID:          "report",
		Name:        "Weekly Report",
		Description: "Weekly status report with summary, accomplishments, blockers and plan",
		Category:    "business",
		Content: `# Weekly Report

**Author:** [Your Name]
**Week:** 2026-W01 (Dec 30 – Jan 3)
**Team:** [Team Name]

---

## Summary

One paragraph overview of the week. Highlight the most important
accomplishment or decision.

## Accomplishments

### Completed

- [x] Shipped feature X to production (PR #123)
- [x] Resolved 3 critical bugs from QA backlog
- [x] Completed design review for feature Y

### In Progress

- [ ] API v2 migration — 70% complete
- [ ] Performance optimization for search — profiling done, implementing fixes

### Metrics

| Metric | This Week | Last Week | Trend |
|--------|-----------|-----------|-------|
| Deploy count | 4 | 3 | ↑ |
| Open bugs | 12 | 15 | ↓ |
| Test coverage | 78% | 76% | ↑ |
| API latency (p99) | 120ms | 145ms | ↓ |

## Blockers

1. **External API dependency** — Waiting on vendor for updated SDK.
   - Impact: Blocks feature Z integration
   - ETA from vendor: Jan 7
2. **CI pipeline instability** — Flaky test in E2E suite causing ~20% failure rate.
   - Workaround: Manual re-runs

## Risks

- Scope creep on feature Y — need to lock requirements by EOW
- Team capacity reduced next week (2 people OOO)

## Next Week Plan

1. Complete API v2 migration
2. Start feature Y implementation
3. Fix CI flaky tests
4. Prepare Q1 planning materials

## Notes

- Shoutout to **Bob** for the excellent post-mortem on last week's incident
- Team offsite scheduled for Jan 15

---

*Generated with MD Editor*
`,
	},
	{
		ID:          "letter",
		Name:        "Letter / Correspondence",
		Description: "Formal letter format for professional correspondence",
		Category:    "writing",
		Content: `[Your Name]
[Your Address Line 1]
[Your Address Line 2]
[City, State/Province, Postal Code]
[Email Address]
[Phone Number]

---

**Date:** January 1, 2026

[Recipient Name]
[Recipient Title]
[Organization Name]
[Address Line 1]
[City, State/Province, Postal Code]

---

**Subject:** [Brief subject of the letter]

Dear [Mr./Ms./Dr. Last Name],

I am writing to [state the purpose of the letter clearly in the opening
paragraph]. This letter is in reference to [provide context or background].

In the second paragraph, provide the details that support your purpose.
Include relevant facts, dates, reference numbers, or any information the
recipient needs to understand or act on your request. Be specific and
concise.

If additional context is needed, use a third paragraph to elaborate. You
may include:

- Supporting point one
- Supporting point two
- Supporting point three

In closing, restate your request or desired outcome. Indicate any deadlines
or next steps, and express your willingness to provide additional information
if needed.

Thank you for your time and consideration. I look forward to hearing from
you at your earliest convenience.

Sincerely,

[Your Signature]

[Your Typed Name]
[Your Title, if applicable]

---

**Enclosures:** [List any attached documents]
**CC:** [Names of anyone receiving copies]
`,
	},
}

// templatesHandler serves built-in markdown templates.
type templatesHandler struct{}

func newTemplatesHandler() *templatesHandler {
	return &templatesHandler{}
}

// templateIndex maps template IDs to their index for O(1) lookup.
var templateIndex map[string]int

func init() {
	templateIndex = make(map[string]int, len(builtinTemplates))
	for i, t := range builtinTemplates {
		templateIndex[t.ID] = i
	}
}

// GET /api/templates — list all templates (without content).
func (h *templatesHandler) list(w http.ResponseWriter, r *http.Request) {
	summaries := make([]TemplateSummary, len(builtinTemplates))
	for i, t := range builtinTemplates {
		summaries[i] = TemplateSummary{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"templates": summaries,
		"count":     len(summaries),
	})
}

// GET /api/templates/{id} — get a single template with full content.
func (h *templatesHandler) get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	idx, ok := templateIndex[id]
	if !ok {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}
	writeJSON(w, http.StatusOK, builtinTemplates[idx])
}
