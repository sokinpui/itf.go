# Developer Guide

This guide is for developers who want to contribute to `itf`.

## Project Structure

The project is organized into several packages:

-   `cmd/itf/`: The main entry point of the application.
-   `cli/`: Command-line interface setup using `cobra`.
-   `itf/`: The core application logic and public API.
-   `internal/`: Internal packages that are not part of the public API.
    -   `fs/`: Filesystem utilities.
    -   `nvim/`: Neovim client and interaction logic.
    -   `parser/`: Markdown parsing and execution plan creation.
    -   `patcher/`: Diff parsing, correction, and application.
    -   `source/`: Logic for reading from clipboard or stdin.
    -   `state/`: Undo/redo history management.
    -   `tui/`: Terminal user interface using `bubbletea`.
-   `model/`: Data structures used across the application.
-   `docs/`: Project documentation.

## Development Setup

1.  **Prerequisites**:
    -   Go 1.21 or later.
    -   Neovim (for running the application).
    -   `patch` command-line tool.

2.  **Clone the repository**:
    ```bash
    git clone https://github.com/sokinpui/itf.go.git
    cd itf.go
    ```

3.  **Install dependencies**:
    ```bash
    go mod tidy
    ```

4.  **Build the binary**:
    ```bash
    go build ./cmd/itf
    ```

    You can now run the tool using `./itf`.

## Running Tests

Currently, the project lacks a comprehensive test suite. Contributions in this area are highly welcome.

When adding tests, please follow the standard Go testing conventions. Place test files next to the code they are testing, with a `_test.go` suffix.

## Contribution Guidelines

1.  **Fork the repository** on GitHub.
2.  **Create a new branch** for your feature or bug fix.
3.  **Write your code**. Please adhere to standard Go formatting and style (`gofmt`).
4.  **Add tests** for your changes, if applicable.
5.  **Ensure your code builds** successfully.
6.  **Submit a pull request** with a clear description of your changes.

Thank you for contributing!
