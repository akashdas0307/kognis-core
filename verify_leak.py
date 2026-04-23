import subprocess
import time
import os
import socket
import signal
from pathlib import Path

PROJECT_ROOT = Path(__file__).parent.resolve()
CORE_DIR = PROJECT_ROOT / "core"
DAEMON_BIN = CORE_DIR / "bin" / "kognis"
SOCKET_PATH = "/tmp/kognis.sock"

def check_port(host, port):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        return s.connect_ex((host, port)) == 0

def get_processes(pattern):
    result = subprocess.run(["pgrep", "-f", pattern], capture_output=True, text=True)
    pids = result.stdout.strip().split("\n")
    return [p for p in pids if p]

def main():
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)
        
    print(f"Starting daemon: {DAEMON_BIN}")
    # Start in a new process group to simulate robust fix later
    process = subprocess.Popen(
        [str(DAEMON_BIN)],
        cwd=CORE_DIR,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    print(f"Daemon PID: {process.pid}")
    
    # Wait for ready
    timeout = 10.0
    start_time = time.time()
    ready = False
    while time.time() - start_time < timeout:
        if os.path.exists(SOCKET_PATH) and check_port("127.0.0.1", 4222):
            ready = True
            break
        if process.poll() is not None:
            print("Daemon died prematurely")
            break
        time.sleep(0.1)
        
    if not ready:
        print("Daemon failed to start")
        process.kill()
        return

    print("Daemon is ready. Checking processes...")
    k_pids = get_processes("bin/kognis")
    print(f"Kognis PIDs: {k_pids}")
    
    # Get initial python processes to avoid false positives
    initial_python = set(get_processes("python"))
    
    # Check for children
    result = subprocess.run(["ps", "-opid", "--no-headers", "--ppid", str(process.pid)], capture_output=True, text=True)
    child_pids = result.stdout.strip().split("\n")
    child_pids = [p for p in child_pids if p]
    print(f"Direct children of daemon: {child_pids}")
    for p in child_pids:
        subprocess.run(["ps", "-fp", p])

    # Check for anything else that might have "nats" (though embedded)
    # Actually, let's see the process tree
    subprocess.run(["ps", "--forest", "-g", str(os.getpgid(os.getpid()))])

    print("Terminating daemon with process.terminate()...")
    process.terminate()
    try:
        process.wait(timeout=5.0)
        print("Daemon terminated gracefully.")
    except subprocess.TimeoutExpired:
        print("Daemon did not terminate, killing...")
        process.kill()
        process.wait()

    time.sleep(2) # Wait a bit for everything to settle
    
    print("Checking for leaks...")
    leaks = get_processes("bin/kognis")
    # Filter out our own script if it matches (it shouldn't match bin/kognis)
    leaks = [p for p in leaks if int(p) != process.pid]
    
    if leaks:
        print(f"LEAK DETECTED (kognis): {leaks}")
    else:
        print("No leaks detected for bin/kognis.")

    current_python = set(get_processes("python"))
    python_leaks = current_python - initial_python
    # Filter out our own PID
    python_leaks = {p for p in python_leaks if int(p) != os.getpid()}
    
    if python_leaks:
        print(f"LEAK DETECTED (python): {python_leaks}")
        for p in python_leaks:
            subprocess.run(["ps", "-fp", p])
    else:
        print("No python leaks detected.")
        
    # Check port 4222
    if check_port("127.0.0.1", 4222):
        print("LEAK DETECTED: Port 4222 still in use!")
        subprocess.run(["ss", "-tulnp", "|", "grep", "4222"], shell=True)
    else:
        print("Port 4222 is clear.")

if __name__ == "__main__":
    main()
