# Config Defaults Simplification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove duplicated in-code config defaults so the example config file is the single complete source of configuration values.

**Architecture:** Keep `config.Load` as a small key=value parser with zero implicit defaults. Parsing fills a zero-value `Config`, applies file values, and returns an error when required settings are missing. Add focused tests around full parsing and missing required fields.

**Tech Stack:** Go 1.21, standard library testing

---

### Task 1: Lock current/desired loader behavior with tests

**Files:**
- Create: `src/config/config_test.go`
- Modify: `src/config/config.go`

**Step 1: Write the failing test**
Add tests for successful parsing from a complete config file and failure when required fields are omitted.

**Step 2: Run test to verify it fails**
Run: `go test ./src/config`
Expected: FAIL because `Load` still injects defaults and does not reject missing fields.

### Task 2: Simplify the loader

**Files:**
- Modify: `src/config/config.go`

**Step 1: Write minimal implementation**
Remove hard-coded defaults, keep parsing logic small, and validate required fields before returning.

**Step 2: Run test to verify it passes**
Run: `go test ./src/config`
Expected: PASS

### Task 3: Align docs and example config

**Files:**
- Modify: `config/config.yaml.example`
- Modify: `README.md`

**Step 1: Update wording only where needed**
Clarify that the file is key=value config and serves as the complete example config.

**Step 2: Run project tests**
Run: `go test ./...`
Expected: PASS
