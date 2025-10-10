Clue can now delegate work to its subagents! Well, at least with search-related tasks.

The development behind this was quite funny:

I initially just wanted to write a bash tool for the agent to execute command, but I completed writing it faster than I expected :) So I thought of improving the search tool that can invoke a subagent.

I was so impressed by [Amp's subagents](https://ampcode.com/agents-for-the-agent) that I tried to copy it (TLDR: Subagents save your main agent's context window and can be invoked by including "subagent" in the prompt - How cool is that?). Think of it like mini-Clues: They should have their own context window and set of tools, and they should be reporting to the big agent in the room.

My design is to have a smaller version of the Agent struct without the persisting conversation slice + MCP connects (will have in the future?), but AI suggests that I use a runner and re-use the same Agent struct so to avoid import cycle with the `tools` package.

But I don't think that will help in the long run. I have an idea of the Maestro mode (cool name, right?) in which the main agent will orchestrate the subagents to finish tasks. The tasks are part of a plan broken down, so each subagent using lightweight model (like Haiku for Claude) can take up a task and finish it. That will save A LOT OF tokens for the main agent's context window and, I hope, keep the agent much longer.

Here is a list of what I do:

1. Made a `subagent.go` file and pretty much copied the inference logic inside `agent.go`. The subagents share the same for loop that will continue invoking the tools until it is done with the task it is given. At that point, it sends back the result to the main agent.
2. Invoke the subagent as a tool. For now, when the main agent invokes the `codebase_search_agent` tool, the subagent will be invoked as a local tool call. For that reason, there is no search function inside `codebase_search_agent.go`.

In the future, I think the approach to separate the logic for subagents is still a good approach. We might get a hoarde of subagents running under the command of the big boss!
