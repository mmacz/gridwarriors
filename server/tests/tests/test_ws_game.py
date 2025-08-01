import asyncio
import pytest
import json
import websockets
import re

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

@pytest.mark.asyncio
async def test_start_game(go_server, log_capture):
    port = go_server.port
    ws_url = f"ws://localhost:{port}/ws"

    async with websockets.connect(ws_url) as ws1, websockets.connect(ws_url) as ws2:
        await ws1.send(json.dumps({
            "type": "join",
            "data": { "name": "PlayerOne" }
        }))
        await ws2.send(json.dumps({
            "type": "join",
            "data": { "name": "PlayerTwo" }
        }))

        await asyncio.sleep(0.2)

        await ws1.send(json.dumps({
            "type": "start",
            "data": {}
        }))

        await asyncio.sleep(0.2)

        pattern = re.compile(r"\[Game \d+\] Started between PlayerOne \(.\) and PlayerTwo \(.\) \| Turn: .")
        assert any(pattern.search(line) for line in log_capture.lines), "Start game log not found"

@pytest.mark.asyncio
async def test_game_start_message(go_server):
    uri = f"ws://localhost:{go_server.port}/ws"

    async with websockets.connect(uri) as ws1, websockets.connect(uri) as ws2:
        await ws1.send(json.dumps({
            "type": "join",
            "data": { "name": "Alice" }
        }))
        await ws2.send(json.dumps({
            "type": "join",
            "data": { "name": "Bob" }
        }))

        await asyncio.sleep(0.2)

        await ws1.send(json.dumps({ "type": "start" }))

        start_msgs = []
        for ws in (ws1, ws2):
            raw = await ws.recv()
            msg = json.loads(raw)
            start_msgs.append(msg)

        types = {m["type"] for m in start_msgs}
        assert types == {"game_start"}, f"Expected game_start messages, got: {types}"

        for m in start_msgs:
            assert "your_role" in m["data"]
            assert m["data"]["your_role"] in ("X", "O")
            assert m["data"]["opponent"] in ("Alice", "Bob")
            assert m["data"]["turn"] in ("X", "O")
