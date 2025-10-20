# API (Library Usage)

`itf` can be used as a Go library to integrate its file modification capabilities into other applications.

## Public API

The public API is located in the `itf` package.

### `Apply`

```go
func Apply(content string, config Config) (map[string][]string, error)
