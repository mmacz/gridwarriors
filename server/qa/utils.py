import os
import time
import signal


def find_file_upward(filename, stop_at=".git", start_dir=None):
    if start_dir is None:
        start_dir = os.getcwd()

    current_dir = os.path.abspath(start_dir)

    while True:
        candidate_path = os.path.join(current_dir, filename)
        stop_path = os.path.join(current_dir, stop_at)

        if os.path.isfile(candidate_path):
            return candidate_path

        if os.path.exists(stop_path):
            break

        parent_dir = os.path.dirname(current_dir)
        if parent_dir == current_dir:
            break

        current_dir = parent_dir

    raise FileNotFoundError(f"{filename} not found (searched up to {stop_at})")

class LogCapture:
    def __init__(self):
        self.lines = []

    def add(self, line: str):
        self.lines.append(line)

    def reset(self):
        self.lines.clear()

    def contains(self, text: str, timeout: float = 3.0) -> bool:
        deadline = time.time() + timeout
        while time.time() < deadline:
            if any(text in line for line in self.lines):
                return True
            time.sleep(0.1)
        return False

class ServerState:
    def __init__(self, port: int, pid = None):
        self._port = port
        self.pid = pid

    @property
    def is_running(self) -> bool:
        if self.pid is None:
            raise RuntimeError("Server PID not set. Did the server start?")
        try:
            os.kill(self.pid, 0)
            return True
        except OSError:
            return False

    @property
    def port(self) -> int:
        return self._port

    def kill(self):
        if self.pid is None:
            raise RuntimeError("Server PID not set. Cannot kill.")
        try:
            os.kill(self.pid, signal.SIGTERM)
        except Exception as e:
            print(f"⚠️ Failed to kill server: {e}")

    def __repr__(self):
        status = "running" if self.is_running else "not running"
        return f"<ServerState pid={self.pid} ({status})>"


