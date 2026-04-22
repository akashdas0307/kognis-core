# Kognis Python SDK

The official Python SDK for building Kognis Framework plugins.

## Installation

```bash
cd sdk/python
pip install -e .
```

## Creating a Plugin

A Kognis plugin consists of a `manifest.yaml` and a Python script implementing the `Plugin` class.

### 1. The Manifest (`manifest.yaml`)

```yaml
manifest_version: 1
plugin_id: my_plugin
plugin_name: My Awesome Plugin
version: 0.1.0
author: Your Name
license: MIT
description: Does something awesome
language: python
handler_mode: stateless
runtime:
  entrypoint: python main.py
slot_registrations: []
```

### 2. The Implementation (`main.py`)

```python
import asyncio
from kognis_sdk.plugin import Plugin
from kognis_sdk.manifest import Manifest

class MyPlugin(Plugin):
    async def on_startup(self) -> None:
        print("Plugin started!")

async def main():
    manifest = Manifest.from_yaml("manifest.yaml")
    plugin = MyPlugin(manifest)
    await plugin.run()

if __name__ == "__main__":
    asyncio.run(main())
```

## Running Examples

The SDK includes a `hello_world` example to get you started.

### Prerequisites

1.  **Kognis Core Daemon:** You must have the Kognis core daemon running. It creates the control plane Unix socket at `/tmp/kognis.sock`.
2.  **NATS Server:** The core daemon usually embeds NATS, but the plugin needs to be able to connect to it (default: `nats://localhost:4222`).

### Running Hello World

```bash
# Navigate to the example directory
cd sdk/python/examples/hello_world

# Run the plugin
python main.py
```

You should see logs indicating the plugin has connected to the control plane, registered, and started emitting health pulses.

## Development

### Running Tests
```bash
cd sdk/python
pytest
```

### Linting
```bash
cd sdk/python
ruff check .
```
