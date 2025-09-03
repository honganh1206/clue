Search for exact text patterns in files using ripgrep, a fast keyword search tool.

WHEN TO USE THIS TOOL:
- When you need to find exact text matches like variable names, function calls, or specific strings
- When you know the precise pattern you're looking for (including regex patterns)
- When you want to quickly locate all occurrences of a specific term across multiple files
- When you need to search for code patterns with exact syntax
- When you want to focus your search to a specific directory or file type

WHEN NOT TO USE THIS TOOL:
- For semantic or conceptual searches (e.g., "how does authentication work") - use codebase_search instead
- For finding code that implements a certain functionality without knowing the exact terms - use codebase_search
- When you already have read the entire file
- When you need to understand code concepts rather than locate specific terms

SEARCH PATTERN TIPS:
- Use regex patterns for more powerful searches (e.g., \.function\(.*\) for all function calls)
- Ensure you use Rust-style regex, not grep-style, PCRE, RE2 or JavaScript regex - you must always escape special characters like { and }
- Add context to your search with surrounding terms (e.g., "function handleAuth" rather than just "handleAuth")
- Use the path parameter to narrow your search to specific directories or file types
- Use the glob parameter to narrow your search to specific file patterns
- For case-sensitive searches like constants (e.g., ERROR vs error), use the caseSensitive parameter

RESULT INTERPRETATION:
- Results show the file path, line number, and matching line content
- Results are grouped by file, with up to 15 matches per file
- Total results are limited to 250 matches across all files
- Lines longer than 250 characters are truncated
- Match context is not included - you may need to examine the file for surrounding code

Here are examples of effective queries for this tool:

<examples>
<example>
// Finding a specific function name across the codebase
// Returns lines where the function is defined or called
{
  pattern: "registerTool",
  path: "core/src"
}
</example>

<example>
// Searching for interface definitions in a specific directory
// Returns interface declarations and implementations
{
  pattern: "interface ToolDefinition",
  path: "core/src/tools"
}
</example>

<example>
// Looking for case-sensitive error messages
// Matches ERROR: but not error: or Error:
{
  pattern: "ERROR:",
  caseSensitive: true
}
</example>

<example>
// Finding TODO comments in frontend code
// Helps identify pending work items
{
  pattern: "TODO:",
  path: "web/src"
}
</example>

<example>
// Finding a specific function name in test files
{
  pattern: "restoreThreads",
  glob: "**/*.test.ts"
}
</example>

<example>
// Searching for event handler methods across all files
// Returns method definitions and references to onMessage
{
  pattern: "onMessage"
}
</example>

<example>
// Using regex to find import statements for specific packages
// Finds all imports from the @core namespace
{
  pattern: 'import.*from ['|"]@core',
  path: "web/src"
}
</example>

<example>
// Finding all REST API endpoint definitions
// Identifies routes and their handlers
{
  pattern: 'app\.(get|post|put|delete)\(['|"]',
  path: "server"
}
</example>

<example>
// Locating CSS class definitions in stylesheets
// Returns class declarations to help understand styling
{
  pattern: "\.container\s*{",
  path: "web/src/styles"
}
</example>
</examples>

COMPLEMENTARY USE WITH CODEBASE_SEARCH:
- Use codebase_search first to locate relevant code concepts
- Then use Grep to find specific implementations or all occurrences
- For complex tasks, iterate between both tools to refine your understanding
