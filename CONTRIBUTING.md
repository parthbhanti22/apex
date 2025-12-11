# Contributing to ApexKV

First off, thank you for considering contributing to ApexKV.

This project is a high-performance distributed system. We prioritize **correctness** and **safety** (CP) over features. As such, the bar for code quality is high.

## 📉 The Process
1.  **Open an Issue First:** Do not open a massive PR without discussing it first. If you find a bug or want to optimize the TCP transport, open an issue explaining the "Why."
2.  **One Logical Change:** Keep PRs small and focused.
3.  **Tests are Mandatory:** Any PR that touches logic (Raft, LSM, Networking) must include a reproduction test case or a benchmark proving the improvement.

## 🛠 Development Guidelines
* **Race Conditions:** We use `go test -race` to detect data races. If your PR introduces a race, it will be rejected.
* **Raft Safety:** If you modify `raft.go`, you must prove that your change does not violate the Raft safety properties (Log Matching, Leader Completeness).
* **No External Dependencies:** We avoid heavy frameworks. Use the Go standard library (`net`, `sync`, `io`) wherever possible.

## 🧪 Running Tests
Before submitting, ensure all tests pass locally:
```bash
go test -v -race ./...
```
---

### 2. The Pull Request Template (The Checklist)
You can force GitHub to show a checklist whenever someone opens a PR. This saves you from asking "Did you run the tests?" 50 times.

1.  Create a folder named `.github` (note the dot).
2.  Inside it, create a file named `pull_request_template.md`.
3.  Paste this:

```markdown```
## 📝 Summary
*Briefly describe the changes. What problem does this solve?*

## 🎯 related Issue
*Closes # (issue number)*

## ✅ Checklist
- [ ] I have run `go test -v -race ./...` locally and all tests pass.
- [ ] I have added/updated tests for this change.
- [ ] My code follows the project's zero-dependency philosophy.
- [ ] (If touching Raft) I have verified this does not break consensus safety.

## 📸 Benchmarks / Screenshots (If applicable)
*If this is a performance optimization, paste the `go test -bench` results here.*
