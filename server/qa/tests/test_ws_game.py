import asyncio
import pytest
import json
import websockets

WS_URL = "ws://localhost:8080/ws"

@pytest.mark.asyncio
async def test_join_and_leave(go_server):
    async with websockets.connect(WS_URL) as ws:
        await ws.send(json.dumps({
            "type": "join",
            "data": {"name": "test_user_1"}
        }))
        await ws.send(json.dumps({
            "type": "leave",
            "data": {}
        }))
