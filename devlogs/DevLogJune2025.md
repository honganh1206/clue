(This should have been done a month ago but I was lazy lol)

- I learn a lot about [Delta vs snapshot streaming](https://docs.anthropic.com/en/docs/build-with-claude/streaming#delta-vs-snapshot-streaming) when intergrating Claude and Gemini into the agent

- At some point I was wondering about handling the streaming response from the model. Maybe a buffer is enough? Get the stream, push the data from the stream to the buffer, then send the data from the buffer to a channel of CustomResponse? I will try to keep it simple first. Maybe using an iterator is a better idea.

- `ContentBlockUnion` as a unified struct for different content block types. Stole this idea from "anthropic-sdk-go" although the implementation is kinda meh

- I have always been thinking of implementing a server to handle all the CRUD-related stuff, just like what Ollama did with the models. And if there will be a server, there should be a database, and sqlite3 is perfect as a lightweight option to store conversations

- A good piece of advice: Too many tools and the agent would get stuck and not know which one to use. A curated set of tools is the most important thing.

- The tokens must flow. The agent should retry the operation instead of halting it

- Ideas for the next tool(s): Commit Diff Lookup
