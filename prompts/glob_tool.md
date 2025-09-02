Fast file pattern matching tool that works with any codebase size

Use this tool to find files by name patterns across your codebase. It returns matching file paths sorted by recent modification time.

## When to use this tool

- When you need to find specific file types (e.g., all JavaScript files)
- When you want to find files in specific directories or following specific patterns
- When you need to explore the codebase structure quickly
- When you need to find recently modified files matching a pattern

## File pattern syntax

- `**/*.js` - All JavaScript files in any directory
- `src/**/*.ts` - All TypeScript files under the src directory (searches only in src)
- `*.json` - All JSON files in the current directory
- `**/*test*` - All files with "test" in their name
- `web/src/**/*` - All files under the web/src directory
- `**/*.{js,ts}` - All JavaScript and TypeScript files (alternative patterns)
- `src/[a-z]*/*.ts` - TypeScript files in src subdirectories that start with lowercase letters

Here are examples of effective queries for this tool:

<examples>
<example>
// Finding all TypeScript files in the codebase
// Returns paths to all .ts files regardless of location
{
  filePattern: "**/*.ts"
}
</example>

<example>
// Finding test files in a specific directory
// Returns paths to all test files in the src directory
{
  filePattern: "src/**/*test*.ts"
}
</example>

<example>
// Searching only in a specific subdirectory
// Returns all Svelte component files in the web/src directory
{
  filePattern: "web/src/**/*.svelte"
}
</example>

<example>
// Finding recently modified JSON files with limit
// Returns the 10 most recently modified JSON files
{
  filePattern: "**/*.json",
  limit: 10
}
</example>

<example>
// Paginating through results
// Skips the first 20 results and returns the next 20
{
  filePattern: "**/*.js",
  limit: 20,
  offset: 20
}
</example>
</examples>

Note: Results are sorted by modification time with the most recently modified files first.
