## Reduce input token

I need a way to reduce the extremely high number of input tokens (think of >100k tokens PER REQUEST) when converting custom message structures from/to SDK-specific structures.

For that I might need to do _intelligent context management_. There are different methods to achieve this:

1. Sliding window: We keep the last 10 messages (configurable?) messages or around 100k tokens? This will lead to context loss from the beginning of the conversation.
2. Summarization: We call a cheaper agent to "compress" the messages after 5-10 messages

We also have to do some _content reduction strategies_:

1. Truncation: The last-resort method. We remove the oldest messages (except the system prompts) one by one until the context fits.
2. Filtering & Pruning: We can shorten the tool output/result that is passed back to the model.

And there is this _prompt steering problem_: Whenever I try to tell the agent to "read file x", it takes it quite literally by reading the absolute path of the file I gave it, then later realizing that there is not such file in that folder, it invokes the search tool.

I need it to invoke the search tool _in the first place_.

Sep 9th: I have just realized that I did not skip the `.git` folder when invoking `list_files` tool :) That's why the number of input tokens was so damn high.

Anyways, the `TruncateMessage` and `SummarizeHistory` methods for `LLMClient` might still be useful in the future.

## Basic TUI

Instead of charmbracelet's libraries for UI, I decided to go minimal with tview (Think of the lightweightness! - I presume).

The question is how do I hook up all the components and streamline the input/output token flows?

Before handling the streaming output token on the TUI, I must first implement a unified output token streaming for all LLM clients (`chatgpt-ui` and `ai-commit` have already handled that quite well). Then it should be easier to hook up the streaming to the TUI?

And another decision to make: Use a string builder to accumulate text or go all-in (a bit overengineered?) with goroutines and channels? It's best to go with option #1 first and make sure it works well before going with #2.
