import asyncio
import pytest
import json
import websockets

WS_URL = "ws://localhost:8080/ws"

@pytest.mark.asyncio
async def test_join_logs(go_server, log_capture):
    async with websockets.connect("ws://localhost:8080/ws") as ws:
        await ws.send(json.dumps({
            "type": "join",
            "data": { "name": "test_user_1" }
        }))

        assert log_capture.contains("Player joined: test_user_1"), "Join log not found"
