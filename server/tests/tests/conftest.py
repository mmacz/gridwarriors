import sys
import os
import subprocess
import pytest
import socket
import time
import threading
import atexit
from random import randrange

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
from utils import (
    find_file_upward,
    LogCapture,
    ServerState
)

proc = None

def is_port_open(host, port):
    with socket.socket() as s:
        return s.connect_ex((host, port)) == 0

def cleanup_server():
    global proc
    if proc and proc.poll() is None:
        print("🧹 Killing lingering Go server...")
        proc.terminate()
        try:
            proc.wait(timeout=2)
        except subprocess.TimeoutExpired:
            proc.kill()
        proc = None

atexit.register(cleanup_server)

@pytest.fixture(scope="function")
def log_capture():
    return LogCapture()

@pytest.fixture(scope="function")
def go_server(log_capture):
    global proc

    main_path = find_file_upward("main.go")
    subprocess.run(["go", "build", "-o", "server_bin", main_path], check=True)

    while True:
        port = randrange(8080, 12400)
        if not is_port_open("localhost", port):
            break

    proc = subprocess.Popen(
        ["./server_bin", "--port", f"{port}"],
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT
    )
    server_state = ServerState(port, proc.pid)
    assert server_state.pid is not None, "Server didn't start"
    print(f"🟢 Starting Go server: localhost:{port} with PID: {server_state.pid}")

    def _read_stdout(pipe):
        for line in iter(pipe.readline, b""):
            decoded = line.decode("utf-8").strip()
            log_capture.add(decoded)

    threading.Thread(target=_read_stdout, args=(proc.stdout,), daemon=True).start()

    timeout = 5
    start = time.time()
    while not is_port_open("localhost", port):
        if time.time() - start > timeout:
            cleanup_server()
            raise RuntimeError("Go server failed to start")
        time.sleep(0.2)

    yield server_state
    assert server_state.is_running
    os.remove("server_bin")

