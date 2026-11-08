# Contributing to dockertrim

Thank you for your interest in contributing to **dockertrim**!

## How to Contribute

1. **Reporting Issues**: Use GitHub Issues to report bugs or request features. Please include logs from `dockertrim.log` if reporting a runtime error.
2. **Pull Requests**:
   - Fork the repository and create a feature branch (`git checkout -b feature/your-feature`).
   - Ensure all unit tests pass: `go test -v ./...`.
   - Ensure the code is formatted: `go fmt ./...`.
   - Submit a Pull Request targeting `main`.

## Safety Guidelines

- Safety first: do not remove safeguards on volume removal or container termination.
- Always provide clear user confirmation before irreversible actions.
