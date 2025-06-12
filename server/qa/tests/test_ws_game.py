import asyncio
import pytest
import json
import websockets

WS_URL = "ws://localhost:8080/ws"

@pytest.mark.asyncio
async def test_join_logs(go_server, log_capture):
    async with websockets.connect(f"ws://localhost:{go_server.port}/ws") as ws:
        await ws.send(json.dumps({
            "type": "join",
            "data": { "name": "test_user_1" }
        }))

        assert log_capture.contains("Player joined: test_user_1"), "Join log not found"

@pytest.mark.asyncio
async def test_multiple_users_connect_and_disconnect(go_server, log_capture):
    users = ["alice", "bob", "charlie"]
    conns = []

    async def connect_user(name):
        ws = await websockets.connect(f"ws://localhost:{go_server.port}/ws")
        await ws.send(json.dumps({
            "type": "join",
            "data": {"name": name}
        }))
        conns.append((name, ws))

    await asyncio.gather(*(connect_user(u) for u in users))
    await asyncio.sleep(0.5)

    for name, _ in conns:
        assert log_capture.contains(f"Player joined: {name}")

    for name, ws in conns:
        await ws.close()
    await asyncio.sleep(0.5)

    for name, _ in conns:
        assert log_capture.contains(f"Player left: {name}")
