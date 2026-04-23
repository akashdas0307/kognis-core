import os
import sys
import subprocess
import time
import socket
import signal
import pytest
from pathlib import Path

# Add project root to path so we can import from tests
PROJECT_ROOT = Path(__file__).parent.parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

from tests.e2e.fixtures.mock_plugins import *

# Constants
CORE_DIR = PROJECT_ROOT / "core"
DAEMON_BIN = CORE_DIR / "bin" / "kognis"
SOCKET_PATH = "/tmp/kognis.sock"


def build_daemon():
    """Build the Go core daemon if it doesn't exist or is requested."""
    print(f"Building Kognis daemon in {CORE_DIR}...")
    result = subprocess.run(
        ["make", "build"],
        cwd=CORE_DIR,
        capture_output=True,
        text=True
    )
    if result.returncode != 0:
        raise RuntimeError(f"Failed to build Kognis daemon:\nSTDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}")
    print("Daemon built successfully.")


@pytest.fixture(scope="session", autouse=True)
def kognis_daemon():
    """
    Session-scoped fixture to start the Kognis core daemon.
    Uses a readiness probe to wait for the Unix socket before yielding.
    """
    # Ensure any stale socket is removed
    if os.path.exists(SOCKET_PATH):
        try:
            os.remove(SOCKET_PATH)
        except OSError:
            pass

    # Build the daemon first
    build_daemon()

    if not DAEMON_BIN.exists():
        raise FileNotFoundError(f"Daemon binary not found at {DAEMON_BIN}")

    # Start the daemon process in a new process group
    print(f"Starting Kognis daemon: {DAEMON_BIN}")
    process = subprocess.Popen(
        [str(DAEMON_BIN)],
        cwd=CORE_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        start_new_session=True
    )

    # Readiness probe: wait for the Unix socket and NATS port to be ready
    ready = False
    socket_ready = False
    nats_ready = False
    timeout = 10.0  # seconds
    start_time = time.time()
    
    def check_port(host, port):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            return s.connect_ex((host, port)) == 0
    
    while time.time() - start_time < timeout:
        if not socket_ready and os.path.exists(SOCKET_PATH):
            socket_ready = True
            
        if not nats_ready and check_port("127.0.0.1", 4222):
            nats_ready = True
            
        if socket_ready and nats_ready:
            time.sleep(0.1)
            ready = True
            break
        
        if process.poll() is not None:
            stdout, stderr = process.communicate()
            raise RuntimeError(f"Daemon crashed during startup.\nSTDOUT:\n{stdout}\nSTDERR:\n{stderr}")
            
        time.sleep(0.1)

    if not ready:
        process.terminate()
        stdout, stderr = process.communicate()
        raise TimeoutError(f"Daemon did not create socket {SOCKET_PATH} or NATS port 4222 within {timeout}s.\nSTDOUT:\n{stdout}\nSTDERR:\n{stderr}")

    print("Kognis daemon is ready.")
    
    yield process

    # Teardown
    print("\nTerminating Kognis daemon process group...")
    try:
        os.killpg(os.getpgid(process.pid), signal.SIGTERM)
        process.wait(timeout=5.0)
    except subprocess.TimeoutExpired:
        print("Daemon did not terminate gracefully, killing process group...")
        try:
            os.killpg(os.getpgid(process.pid), signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.wait()
    except ProcessLookupError:
        pass
        
    # Clean up socket
    if os.path.exists(SOCKET_PATH):
        try:
            os.remove(SOCKET_PATH)
        except OSError:
            pass
