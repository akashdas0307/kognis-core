---
name: kognis-graphify-workflow
description: Integration of Graphify for planning, research, and post-implementation synchronization.
---
# Graphify Workflow Rule

- **Planning/Research**: Use `/graphify query "[topic]"` to map architectural dependencies and spec-implementation consistency before starting a new milestone.
- **Context Management**: Prioritize graph-based navigation for high-level questions to reduce token usage.
- **Synchronization**: Run `/graphify --update` immediately following the completion of any "Hard Milestone" or Phase boundary to keep the knowledge graph current.
- **Reporting**: Reference the `graphify-out/GRAPH_REPORT.md` in Phase summaries to provide structural evidence of system integrity.
