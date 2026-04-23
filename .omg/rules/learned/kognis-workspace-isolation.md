---
name: kognis-workspace-isolation
description: Management of workspace lanes and registry path dependencies.
---
# Workspace & Registry Rule

- **Workspace State**: Maintain `.omg/state/workspace.json` as the source of truth for execution lanes and branch anchors.
- **Lanes**: Always assign an explicit `owner` (agent role) and `purpose` to each active lane.
- **Registry Integration**: Treat the sibling directory `../kognis-registry` (or the external repo `https://github.com/akashdas0307/kognis-registry.git`) as the authoritative registry for external plugin schemas during E2E testing.
- **Cleanliness**: Ensure all temporary state files (e.g., OM/SDK checkpoints) are excluded from feature commits while maintaining their presence for autonomous resumption.
