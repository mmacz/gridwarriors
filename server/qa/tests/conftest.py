import sys 
import os
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

import subprocess
import time
import pytest
import socket
from utils import find_file_upward

def is_port_open(host, port):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        return s.connect_ex((host, port)) == 0

@pytest.fixture(scope="session")
def go_server():
    print(">>>> Starting GridWarriors server")
    proc = subprocess.Popen(["go", "run", find_file_upward("main.go")])
    timeout = 5
    start = time.time()
    while not is_port_open("localhost", 8080):
        if time.time() - start > timeout:
            proc.kill()
            raise RuntimeError("Server didn't start")
        time.sleep(0.2)

    yield
    proc.terminate()

