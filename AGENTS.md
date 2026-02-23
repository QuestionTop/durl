# AI General Code of Conduct

## Behavioral Guidelines

**Core Principles**: Caution over speed, code simplicity, goal-driven.

### 1. Think Before Coding
- State assumptions clearly, ask if uncertain
- Explain when there are multiple interpretations, don't silently choose
- Speak up if there's a simpler approach
- Stop and clarify confusion points

### 2. Simplicity First
- Only implement required features, no speculative features
- Don't write abstractions for single-use scenarios
- No unrequested "flexibility" or "configurability"
- Rewrite if 200 lines can be 50 lines

### 3. Precise Modifications
- Only modify what's necessary
- Don't "optimize" adjacent code, comments, or formatting
- Don't refactor code that works fine
- Match existing style
- Only clean up orphaned code you created

### 4. Goal-Driven
- Define verifiable success criteria
- For multi-step tasks, explain the plan: `[Step] → Verify: [Check]`

---

## General Coding Constraints

1. **Do not generate usage documentation**
2. Run tests after generating code
3. Delete test files after use

---

## Shell Command Guidelines

### PowerShell Guidelines
- Paths with spaces must use double quotes: `mkdir "/path/with spaces"`
- Multi-line commands use backtick for continuation: `command1 \`
- Does not support Unix flags, use PowerShell parameters: `ls -Force` to show hidden files
- Don't use `Get-Content "file.txt" | Select-Object -First 100` type grep operations, use `rg` tool instead
- Delete can use `Remove-Item -Path "file.txt" -Force` instead of `rm` commands

### Git Operations
- Must add `-c core.autocrlf=false` parameter
- Example: `git -c core.autocrlf=false add .`

### Interactive Commands
When generating commands for automatic execution, do not include interactive commands. For example, if psql requires password input, it should be set in command line arguments in advance. Assuming database connection password is 1:
- **bash environment**: `export PGPASSWORD='1' && psql -U postgres -d report`
- **PowerShell environment**: `$env:PGPASSWORD='1';psql -U postgres -d report`

---

## Project Guidelines

**This is an international-facing project.**

- All code comments and documentation must be in English
- All files must use UTF-8 encoding
- Follow international coding standards and best practices

### Build & Versioning

- Always build with version injected via ldflags:
  ```
  go build -ldflags "-X main.version=<tag>" -o durl.exe .
  ```
- Plain `go build .` produces a `dev` build (version = "dev")
- To release a new version (PowerShell):
  ```powershell
  $env:VER="v<X.Y>"
  git -c core.autocrlf=false tag $env:VER
  go build -ldflags "-X main.version=$env:VER" -o durl.exe .
  git push origin $env:VER
  gh release create $env:VER durl.exe --title $env:VER --notes "Release $env:VER"
  ```

### Adding a New --site Plugin

When adding a new `--site=<name>` scraper, always do ALL of the following:

1. Create `internal/sites/<name>/` with `scraper.go`, `client.go`, `content.go`
2. Register it in `main.go` with a blank import: `_ "durl/internal/sites/<name>"`
3. **Update the `Example` block in `main.go`** to include example commands for the new site， update info to README.md — no Chinese characters allowed in any example strings
4. Keep all example lines consistently indented with **2 spaces** (not tabs)
