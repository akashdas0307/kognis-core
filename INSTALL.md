# Installation Guide — Kognis Framework

This guide provides instructions for setting up the Kognis Framework on your local machine.

## Prerequisites

- **Go 1.25+** (for the core daemon)
- **Python 3.11+** (for plugins and the SDK)
- **Linux or macOS** (Windows support via WSL2 is possible but not officially tested)
- **NATS Server** (The core daemon embeds NATS, but having the `nats` CLI is helpful for debugging)

## 1. Build the Core Daemon

The core daemon is written in Go and acts as the nervous system of the framework.

```bash
cd core
make build
```

The binary will be created at `core/bin/kognis`.

## 2. Install the Python SDK

The Python SDK is required to run official plugins and build your own.

```bash
cd sdk/python
python -m venv venv
source venv/bin/activate
pip install -e .
```

## 3. Configure the Framework

Kognis looks for a configuration file and a plugins directory. By default, it uses `./core/config.yaml` or flags.

Create a basic `config.yaml` in the `core` directory:

```yaml
nats:
  embedded: true
  port: 4222
supervisor:
  plugins_dir: "../plugins"
  restart_backoff_ms: 1000
```

## 4. Run the Framework

Start the core daemon:

```bash
cd core
./bin/kognis
```

To enable the TUI dashboard:

```bash
./bin/kognis --tui
```

## 5. Running Plugins

Once the core daemon is running, it will automatically discover and spawn plugins located in the `plugins_dir` specified in your config.

To manually run a plugin for development:

```bash
cd sdk/python/examples/hello_world
python main.py
```

## Troubleshooting

- **Socket Errors:** The core daemon creates a Unix socket at `/tmp/kognis.sock`. Ensure the process has permission to write to this location.
- **Port Conflicts:** If port 4222 is already in use by another NATS instance, disable the embedded NATS or change the port in `config.yaml`.
- **Import Errors:** Ensure you have activated your Python virtual environment where the `kognis-sdk` was installed.
