Read a file from the file system. If the file doesn't exist, an error is returned.

- The path parameter must be an absolute path.
- By default, this tool returns the first 1000 lines. To read more, call it multiple times with different read_ranges.
- Use the Grep tool to find specific content in large files or files with long lines.
- If you are unsure of the correct file path, use the glob tool to look up filenames by glob pattern.
- The contents are returned with each line prefixed by its line number. For example, if a file has contents "abc\
", you will receive "1: abc\
".
- This tool can read images (such as PNG, JPEG, and GIF files) and present them to the model visually.
- When possible, call this tool in parallel for all files you will want to read.
