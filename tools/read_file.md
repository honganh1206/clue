Read a file from the file system. If the file doesn't exist, an error is returned.

- When the user provides a specific file path, read it directly without searching first
- By default, this tool returns the first 1000 lines. To read more, call it multiple times with different read_ranges.
- Use the Grep tool to find specific content in large files or files with long lines.
- The contents are returned with each line prefixed by its line number. For example, if a file has contents "abc\
", you will receive "1: abc\
".
- When possible, call this tool in parallel for all files you will want to read.
