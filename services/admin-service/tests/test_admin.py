"""Offline SQLite and authentication tests; copy into telegram-admin/admin-service/tests/."""
import importlib.util
from pathlib import Path
import pytest
from fastapi.testclient import TestClient

ROOT = Path(__file__).resolve().parents[1]

@pytest.fixture
def client(tmp_path, monkeypatch):
    monkeypatch.setenv("DB_PATH", str(tmp_path / "admin.db"))
    monkeypatch.setenv("ADMIN_USER", "tester")
    monkeypatch.setenv("ADMIN_PASSWORD", "test-secret")
    monkeypatch.setenv("EVENT_TOKEN", "event-secret")
    spec = importlib.util.spec_from_file_location("admin_test_app", ROOT / "main.py")
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    # Explicit startup because app uses legacy startup event.
    mod.init()
    with TestClient(mod.app) as c:
        yield c

def auth():
    return ("tester", "test-secret")

def event_headers(token="event-secret"):
    return {"X-Event-Token": token}

def test_admin_endpoints_require_auth(client):
    for path in ("/", "/api/overview", "/api/users", "/api/logs"):
        assert client.get(path).status_code == 401
        assert client.get(path, auth=("tester", "wrong")).status_code == 401

def test_event_endpoint_requires_token(client):
    payload = {"event": "user_activity", "telegram_id": 123}
    assert client.post("/internal/events", json=payload).status_code == 403
    assert client.post("/internal/events", json=payload, headers=event_headers("wrong")).status_code == 403

def test_rejects_invalid_event(client):
    assert client.post("/internal/events", json={}, headers=event_headers()).status_code == 422
    assert client.post("/internal/events", json={"event": "x" * 101}, headers=event_headers()).status_code == 422

def test_event_is_saved_and_user_search_count(client):
    payload = {"event": "anime_search", "telegram_id": 12345, "username": "alice", "detail": "Naruto"}
    assert client.post("/internal/events", json=payload, headers=event_headers()).json() == {"ok": True}
    users = client.get("/api/users", auth=auth()).json()
    logs = client.get("/api/logs", auth=auth()).json()
    overview = client.get("/api/overview", auth=auth()).json()
    assert len(users) == 1
    assert users[0]["telegram_id"] == 12345
    assert users[0]["searches"] == 1
    assert logs[0]["event"] == "anime_search"
    assert overview["users"] == 1

def test_sql_injection_string_is_stored_as_data(client):
    payload = {"event": "user_activity", "telegram_id": 99,
               "username": "x'); DROP TABLE users;--", "detail": "'; DELETE FROM logs;--"}
    assert client.post("/internal/events", json=payload, headers=event_headers()).status_code == 200
    users = client.get("/api/users", auth=auth()).json()
    assert users[0]["username"] == payload["username"]
    assert client.get("/api/logs", auth=auth()).status_code == 200

def test_detail_is_truncated(client):
    assert client.post("/internal/events", headers=event_headers(),
                       json={"event": "test", "detail": "x" * 4000}).status_code == 200
    logs = client.get("/api/logs", auth=auth()).json()
    assert len(logs[0]["detail"]) == 2000

def test_activity_and_settings(client):
    assert client.get('/api/activity').status_code == 401
    assert client.get('/api/settings').status_code == 401
    assert client.put('/api/settings', json={'refresh_seconds': 5, 'activity_days': 7}).status_code == 401
    for uid in [101, 102, 101]:
        assert client.post('/internal/events', headers=event_headers(), json={'event':'user_activity','telegram_id':uid}).status_code == 200
    activity = client.get('/api/activity?days=7', auth=auth()).json()['days']
    assert activity[-1]['active_users'] == 2
    assert activity[-1]['events'] == 3
    assert client.put('/api/settings',auth=auth(),json={'refresh_seconds':2,'activity_days':7}).status_code == 422
    assert client.put('/api/settings',auth=auth(),json={'refresh_seconds':15,'activity_days':14}).status_code == 200
    assert client.get('/api/settings',auth=auth()).json() == {'refresh_seconds':15,'activity_days':14}
