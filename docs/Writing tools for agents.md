[Link](https://www.anthropic.com/engineering/writing-tools-for-agents)

## Choosing the right tools for agents

Make sure _each tool you build has a clear, distinct purpose_.

Tools should enable agents to subdivide and solve tasks in much the same way that a human would.

Too many tools or overlapping tools can also distract agents from pursuing efficient strategies. Careful, selective planning of the tools you build (or don’t build) can really pay off.

## Namespacing your tools

Namespacing (grouping related tools under common prefixes) can help delineate boundaries between lots of tools For example, namespacing tools by service (e.g., asana_search, jira_search) and by resource (e.g., asana_projects_search, asana_users_search), can help agents select the right tools at the right time.

> Selecting between prefix- and suffix-based namespacing to have non-trivial effects on tool-use evaluations.

## Returning meaningful context from your tools

Tool implementations should take care to return only high signal information back to agents.
