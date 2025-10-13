# Installation

There are a few ways to install the `itf` command-line tool.

## Using `go install`

If you have a Go environment set up, you can install `itf` directly from the source repository:

```bash
go install github.com/sokinpui/itf.go/cmd/itf@latest
```

This will download the source, compile it, and place the `itf` binary in your Go bin directory (`$GOPATH/bin` or `$HOME/go/bin`). Make sure this directory is in your system's `PATH`.

## From Source

You can also clone the repository and build the project manually.

```bash
# Clone the repository
git clone https://github.com/sokinpui/itf.go.git

# Navigate to the project directory
cd itf.go

# Build the binary
go build ./cmd/itf

# Move the binary to a directory in your PATH
mv itf /usr/local/bin/
```

## Pre-built Binaries

Pre-built binaries for various operating systems and architectures may be available on the [GitHub Releases](https://github.com/sokinpui/itf.go/releases) page. You can download the appropriate binary for your system, extract it, and place it in a directory included in your `PATH`.
