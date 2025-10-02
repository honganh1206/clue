Executes the given shell command in the user's default shell.

## Important notes

1. Directory verification:
   - If the command will create new directories or files, first use the Read tool to verify the parent directory exists and is the correct location
   - For example, before running a mkdir command, first use Read to check if the parent directory exists

2. Working directory:
   - If no `cwd` parameter is provided, the working directory is the first workspace root folder.
   - If you need to run the command in a specific directory, set the `cwd` parameter to an absolute path to the directory.
   - Avoid using `cd` (unless the user explicitly requests it); set the `cwd` parameter instead.

3. Multiple independent commands:
   - Do NOT chain multiple independent commands with `;`
   - Do NOT chain multiple independent commands with `&&` when the operating system is Windows
   - Do NOT use the single `&` operator to run background processes
   - Instead, make multiple separate tool calls for each command you want to run

4. Escaping & Quoting:
   - Escape any special characters in the command if those are not to be interpreted by the shell
   - ALWAYS quote file paths with double quotes (eg. cat "path with spaces/file.txt")
   - Examples of proper quoting:
     - cat "path with spaces/file.txt" (correct)
     - cat path with spaces/file.txt (incorrect - will fail)

5. Truncated output:
   - Only the last 50000 characters of the output will be returned to you along with how many lines got truncated, if any
   - If necessary, when the output is truncated, consider running the command again with a grep or head filter to search through the truncated lines

6. Stateless environment:
   - Setting an environment variable or using `cd` only impacts a single command, it does not persist between commands

7. Cross platform support:
    - When the Operating system is Windows, use `powershell` commands instead of Linux commands
    - When the Operating system is Windows, the path separator is '``' NOT '`/`'

8. User visibility
    - The user is shown the terminal output, so do not repeat the output unless there is a portion you want to emphasize

9. Avoid interactive commands:
   - Do NOT use commands that require interactive input or wait for user responses (e.g., commands that prompt for passwords, confirmations, or choices)
   - Do NOT use commands that open interactive sessions like `ssh` without command arguments, `mysql` without `-e`, `psql` without `-c`, `python`/`node`/`irb` REPLs, `vim`/`nano`/`less`/`more` editors
   - Do NOT use commands that wait for user input

## Examples

- To run 'go test ./...': use { cmd: 'go test ./...' }
- To run 'cargo build' in the core/src subdirectory: use { cmd: 'cargo build', cwd: '/home/user/projects/foo/core/src' }
- To run 'ps aux | grep node', use { cmd: 'ps aux | grep node' }
- To print a special character like $ with some command `cmd`, use { cmd: 'cmd \$' }

## Git

Use this tool to interact with git. You can use it to run 'git log', 'git show', or other 'git' commands.

When the user shares a git commit SHA, you can use 'git show' to look it up. When the user asks when a change was introduced, you can use 'git log'.

If the user asks you to, use this tool to create git commits too. But only if the user asked.

<git-example>
user: commit the changes
assistant: [uses Bash to run 'git status']
[uses Bash to 'git add' the changes from the 'git status' output]
[uses Bash to run 'git commit -m "commit message"']
</git-example>

<git-example>
user: commit the changes
assistant: [uses Bash to run 'git status']
there are already files staged, do you want me to add the changes?
user: yes
assistant: [uses Bash to 'git add' the unstaged changes from the 'git status' output]
[uses Bash to run 'git commit -m "commit message"']
</git-example>

IMPORTANT notes:

- When possible, combine the "git add" and "git commit" commands into a single "git commit -am" command, to speed things up. However, be careful not to stage files (e.g. with git add .) for commits that aren't part of the change, they may have untracked files they want to keep around, but not commit.
- NEVER update the git config
- DO NOT push to the remote repository
- NEVER use git commands with the -i flag (like git rebase -i or git add -i) since they require interactive input which is not supported.
- If there are no changes to commit (i.e., no untracked files and no modifications), do not create an empty commit
- Ensure your commit message is meaningful and concise. It should explain the important parts of the changes, not just describe them.

## Prefer specific tools

It's VERY IMPORTANT to use specific tools when searching for files, instead of issuing terminal commands with find/grep/ripgrep. Use codebase_search or Grep instead. Use Read tool rather than cat, and edit_file rather than sed.
