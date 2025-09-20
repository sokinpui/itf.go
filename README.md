golang rewrite of python [itf](https://github.com/sokinpui/itf)

# ITF: Insert To File

Tired of copying code from LLM Web interfaces.

Too lazy to copy and paste.

Looking for free AI editor...

```
Usage: itf [flags]

Parse content from stdin (pipe) or clipboard to update files in Neovim.

Example: pbpaste | itf -e py

Flags:
  -b, --buffer               Update buffers in Neovim without saving them to disk (changes are saved by default).
  -e, --extension strings    Filter by extension. Use 'diff' to process only diff blocks (e.g., 'py', 'js', 'diff').
  -l, --lookup-dir strings   Change directory to look for files (default: current directory).
  -o, --output-diff-fix      Print the diff that corrected start and count.
  -r, --redo                 Redo the last undone operation.
  -u, --undo                 Undo the last operation.
```

# Installatoin

```
go install github.com/sokinpui/itf.go/cmd/itf@latest
```

locally:

```
git clone https://github.com/sokinpui/itf.go
cd itf.go
git install ./cmd/itf
```

# Format

## Content file block:

a code block that upper line contains a path

for example:

````
`path/to/some/file.py`

```
print(hello)

```
````

## diff block format:

start with `--- a/path/to/file` and `--- b/path/to/file`, following by `@@ -<old start>,<old count> +<new start><new count> @@`
It doesn't matter if start and count is incorrect. I have never seen AI has generate correct start and count even line number provided

You can ask AI to generate in this format.

```diff
--- a/src/main.py
+++ b/src/main.py
@@ -1,7 +1,8 @@
 import os

 def main():
-    print("Hello from main!")
+    # A new, more welcoming message
+    print("Hello, world! Welcome to ITF.")

 if __name__ == "__main__":
     main()
```
