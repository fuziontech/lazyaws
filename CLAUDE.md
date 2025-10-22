# Claude Development Workflow

This document describes the development workflow for lazyaws when working with Claude or AI assistants.

## Development Cycle

Follow this iterative process to chip away at the TODO.md:

### 1. Pick a Task
- Review `TODO.md` and select a logical chunk of work
- Choose related items that make sense to implement together
- Aim for small, focused PRs (not too large, not too small)

### 2. Implement the Feature
- Write the implementation code
- **Include tests** - Every feature should have tests
- Update any relevant documentation
- Keep commits focused and atomic

### 3. Self-Review
- Review your own changes before committing
- Check for:
  - Code quality and consistency
  - Error handling
  - Edge cases
  - Test coverage
  - Documentation updates

### 4. Commit Changes
- **Do NOT include Claude/AI attribution in commits**
- Write clear, descriptive commit messages
- Include all changes in a single commit (or logical sequence)
- **Update TODO.md** - Mark completed items with `[x]`
- Commit message format:
  ```
  Brief description of what was implemented

  - Bullet point of changes
  - Another change
  - Update TODO.md with completed items
  ```

### 5. Open a Pull Request
- Create a PR with the changes
- PR should include:
  - Clear description of what was implemented
  - Reference to TODO.md items completed
  - Any testing notes or considerations
- Use `gh pr create` command

### 6. Repeat
- Merge the PR
- Pick the next task from TODO.md
- Continue the cycle

## Example Session

```bash
# 1. Pick task: "Add EC2 instance filtering by state"
# 2. Implement feature with tests
# 3. Review changes
git status
git diff

# 4. Commit (update TODO.md first!)
git add .
git commit -m "Add EC2 instance filtering by state

- Implement state filter dropdown
- Add keyboard shortcuts for common filters
- Include unit tests for filter logic
- Update TODO.md with completed items"

# 5. Open PR
gh pr create --title "Add EC2 instance filtering" --body "..."

# 6. Merge and repeat
gh pr merge
```

## Guidelines

### Commit Messages
- **NO Claude/AI attribution** - commits should be clean
- Use present tense ("Add feature" not "Added feature")
- Be descriptive but concise
- Reference TODO.md updates

### Testing
- Every feature needs tests
- Unit tests for business logic
- Integration tests for AWS interactions (mock when possible)
- Run tests before committing: `go test ./...`

### TODO.md Updates
- Always update TODO.md in the same commit/PR
- Mark completed items with `[x]`
- Add new items if discovered during implementation
- Keep the roadmap up to date

### PR Size
- Aim for PRs that take 30-60 minutes to implement
- Too small: combining trivial changes
- Too large: breaking into phases
- Each PR should be a complete, working feature

## Testing Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests verbosely
go test -v ./...

# Build and test
go build && go test ./...
```

## Branch Strategy

- Work on feature branches for larger features
- Can commit directly to main for small fixes
- Always use PRs for review and TODO.md tracking

## Notes

- Keep the development momentum going
- Document as you go
- Update README.md when user-facing features are added
- Have fun building!
