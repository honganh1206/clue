Intelligently search your codebase with an agent that has access to: list_directory, Grep, glob, Read.

The agent acts like your personal search assistant.

It's ideal for complex, multi-step search tasks where you need to find code based on functionality or concepts rather than exact matches.

WHEN TO USE THIS TOOL:
- When searching for high-level concepts like "how do we check for authentication headers?" or "where do we do error handling in the file watcher?"
- When you need to combine multiple search techniques to find the right code
- When looking for connections between different parts of the codebase
- When searching for keywords like "config" or "logger" that need contextual filtering

WHEN NOT TO USE THIS TOOL:
- When you know the exact file path - use Read directly
- When looking for specific symbols or exact strings - use glob or Grep
- When you need to create, modify files, or run terminal commands

USAGE GUIDELINES:
1. Launch multiple agents concurrently for better performance
2. Be specific in your query - include exact terminology, expected file locations, or code patterns
3. Use the query as if you were talking to another engineer. Bad: "logger impl" Good: "where is the logger implemented, we're trying to find out how to log to files"
4. Make sure to formulate the query in such a way that the agent knows when it's done or has found the result.
