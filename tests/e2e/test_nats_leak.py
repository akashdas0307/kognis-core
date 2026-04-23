import subprocess
import time
import pytest

def test_nats_leak(kognis_daemon):
    """
    Test that uses the kognis_daemon fixture and then checks for leaking NATS processes.
    """
    print("\n[test_nats_leak] Daemon is running.")
    # Check if NATS is running while daemon is up
    result = subprocess.run(["pgrep", "-f", "kognis"], capture_output=True, text=True)
    print(f"[test_nats_leak] kognis processes: {result.stdout.strip()}")
    
    result = subprocess.run(["pgrep", "-f", "nats"], capture_output=True, text=True)
    print(f"[test_nats_leak] nats processes: {result.stdout.strip()}")

def test_check_after_teardown():
    """
    This test runs after the session-scoped fixture kognis_daemon (in theory, but it's session-scoped so it won't tear down yet).
    Wait, if I want to check AFTER teardown, I need to either use a function-scoped fixture or check at the end of the session.
    """
    pass

@pytest.fixture(scope="function")
def function_daemon(kognis_daemon):
    # This just uses the session daemon, doesn't help with teardown check
    yield kognis_daemon

@pytest.mark.skip(reason="Conflicts with session-scoped kognis_daemon. Run verify_leak.py instead.")
def test_verify_no_leak():
    # To truly verify leak, I should manually start and stop a daemon in this test
    from tests.e2e.conftest import DAEMON_BIN, CORE_DIR, SOCKET_PATH
    import os
    import socket
    
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)
        
    print(f"\n[test_verify_no_leak] Starting manual daemon: {DAEMON_BIN}")
    process = subprocess.Popen(
        [str(DAEMON_BIN)],
        cwd=CORE_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        start_new_session=True
    )
    
    # Wait for ready
    ready = False
    nats_ready = False
    timeout = 10.0
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        if os.path.exists(SOCKET_PATH):
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                if s.connect_ex(("127.0.0.1", 4222)) == 0:
                    nats_ready = True
                    ready = True
                    break
        time.sleep(0.1)
    
    assert ready, "Daemon failed to start"
    print("[test_verify_no_leak] Daemon is ready. Now terminating...")
    
    import signal
    os.killpg(os.getpgid(process.pid), signal.SIGTERM)
    process.wait(timeout=5.0)
    
    print("[test_verify_no_leak] Daemon terminated. Checking for leaks...")
    
    # Check for kognis leaks
    result = subprocess.run(["pgrep", "-f", "kognis"], capture_output=True, text=True)
    leaks = result.stdout.strip().split("\n")
    leaks = [l for l in leaks if l]
    print(f"Leaking kognis processes: {leaks}")
    
    # Check for nats leaks
    result = subprocess.run(["pgrep", "-f", "nats"], capture_output=True, text=True)
    nats_leaks = result.stdout.strip().split("\n")
    nats_leaks = [l for l in nats_leaks if l]
    print(f"Leaking nats processes: {nats_leaks}")
    
    assert not leaks, f"Found leaking kognis processes: {leaks}"
    assert not nats_leaks, f"Found leaking nats processes: {nats_leaks}"
