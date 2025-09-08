golang rewrite of python itf

# ITF: Insert To File

Tired of copying code from LLM Web interfaces.
Too lazy to paste into multiple files.
Don't have cash for Cursor AI.

```
usage: itf [-h] [-s] [-c] [-o] [-l DIR [DIR ...]] [-e EXT [EXT ...]] [-r | -R] [-f | -d | -a]

Parse clipboard content or 'itf.txt' to update files and load them into Neovim.

options:
  -h, --help            show this help message and exit
  -s, --save            Save all modified buffers in Neovim after the update.
  -c, --clipboard       Parse content from the clipboard instead of 'itf.txt'.
  -o, --output-diff-fix
                        print the diff that corrected start and count
  -l, --lookup-dir DIR [DIR ...]
                        change directory to look for files (default: current directory).
  -e, --extension EXT [EXT ...]
                        Filter to process only files with the specified extensions (e.g., 'py', 'js').
  -r, --revert          Revert the last operation. support undo tree, multiple levels of undo
  -R, --redo            Redo the last reverted operation, support redo tree, multiple levels of redo
  -f, --file            ignore diff blocks, parse content files blocks only.
  -d, --diff            parse only diff blocks, ignore content file blocks.
  -a, --auto            parse both diff blocks and content file blocks.

```

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
