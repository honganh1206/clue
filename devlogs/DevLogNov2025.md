## Planner

Writing a planner for an agent is tougher than I thought.

What's a plan in this context by the way? A list containing a sequence of steps for the agent to follow when doing something. It should be used for big tasks like implementing a new fancy features or figuring out how to solve a nasty bug.

Since the agent is meant to be autonomous, I want it to come up with its own plan, figure out the hairy parts and execute the plan itself. But human in the loop should still be the focus.

Anyways, make it generate a plan to do a specific thing is one thing, but to handle it on the backend? Now I have to design a schema for a plan (which I also shamelessly stolen the design from smolcode), figure out how to save it efficiently, figure out how to retrieve it again when the agent needs it. Lots of lots of familiar yet unfamiliar work.

Dario (creator of smolcode) went for the direct `*sql.DB` connection to the SQLite database inside the tool. But I'm more flowery: Create a `PlanModel` -> Write CRUD methods for the model -> Wrap the `PlanModel` with the `Model` struct -> Write handlers for the plan model -> Write client methods corresponding to the handlers -> Make the `plan_write` tool perform CRUD over the client.

That was... not a good way to design IMO, but still, it helps me learn.

It took me half a week to figure out why I got so many 500 internal server errors. First I thought the issue might be from `SavePlan` doing multiple atomic `INSERT` and `UPDATE` operations in a transaction via the same `*api.Client` object the tool shares with the agent. I underestimate SQLite, thinking that it cannot both update the `plan` + `step` + `step_acceptance_criteria` tables while updating `conversations` + `messages` tables at the same time. That was not the case, and SQLite can handle that with ease.

The true problem is about exporting the struct fields. Turns out my initial design was that I made all the fields of `Step` struct private, so those fields don't get serialized when going back and forth between the server and the client. That's why I got so many failed constraint checks for the `Status` field of the `Step` struct.

It might have cost about $5 dollars of Claude API, plus maybe $10 of Amp code to figure that out. Fair price for a lesson.

Now that the hard part of creating a plan is done, here comes the harder part: How do I update the statuses of the steps? For now I'm thinking of fetching the plan via `plan_read` and make it an in-memory plan object, then the agent should be able to overwrite that in-memory plan object. But that sounds dumb, since the agent then has to make two tool calls, and regardless of how fast the agent can call tools, this approach still adds overhead of calling tools back and forth.

Why can't we just make 1 tool call to `plan_write` to fetch the plan and update it? That would be much more efficient, and also we don't need to read from the plan while modifying it anyways (The main agent should be done with editing the plan when it decides to execute it, or let the subagents handle the steps). Plus it sounds more reasonable.

And let's not talk about the TUI: How do we make sure the TUI displays the plan as its latest version? Are we going to re-fetch the plan whenever there is an update? If so, how do we do so? Are we going to spin up a goroutine that listens to each change (like an event) that requests the client to fetch the latest version? That would be so much requests and response.

-> The solution might be to figure out the way we can _sync-up the in-mem plan and the plan on the DB via events_.

I'm also thinking of adding a `plan_id` column to the `conversations` table, since I'm thinking that each conversation should be associated with a plan, and that plan will be read, created and updated (not removed). For that I need to:

1. Change the current `id` field to `plan_name`, and the `id` of the plan will be automatically generated as a GUID/UUID by the server.
2. Update the `plan_id` of `steps` table to now refer to the PK of the `plans` table.
3. Update the `step_acceptance_criteria` accordingly
