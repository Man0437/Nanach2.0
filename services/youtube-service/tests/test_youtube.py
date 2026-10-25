"""Offline API tests; copy into services/youtube-service/tests/."""
import importlib.util
import sys
import types
from pathlib import Path
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

ROOT = Path(__file__).resolve().parents[1]

@pytest.fixture
def app_module(monkeypatch):
    monkeypatch.setenv(
        "YOUTUBE_SERVICE_TOKEN",
        "test-secret-token"
    )

    monkeypatch.setitem(
        sys.modules,
        "yt_dlp",
        types.SimpleNamespace(YoutubeDL=object)
    )

    spec = importlib.util.spec_from_file_location(
        "youtube_test_app",
        ROOT / "main.py"
    )

    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)

    return mod


@pytest.fixture
def client(app_module):
    with TestClient(
        app_module.app,
        headers={
            "X-Service-Token": "test-secret-token"
        }
    ) as test_client:
        yield test_client

@pytest.mark.parametrize("url", [
    "https://youtube.com/watch?v=abc", "https://www.youtube.com/watch?v=abc",
    "https://youtu.be/abc", "https://m.youtube.com/watch?v=abc",
])
def test_allowed_domains(app_module, url):
    assert app_module.valid_url(url)

@pytest.mark.parametrize("url", [
    "http://youtube.com/watch?v=abc", "https://youtube.com.evil.test/watch?v=abc",
    "https://evil.test/watch?v=abc", "https://127.0.0.1/a",
    "https://localhost/a", "file:///etc/passwd", "https://user:pass@youtube.com/watch?v=x",
    "https://youtube.com:443/watch?v=x", "https://[::1]/",
    "not a URL", "https://youtube.com@evil.test/watch?v=x",
])
def test_reject_unsafe_urls(app_module, url):
    assert not app_module.valid_url(url)

@pytest.mark.parametrize("url", ["https://127.0.0.1/private", "https://youtube.com.evil.test/watch?v=abc"])
def test_endpoint_rejects_bad_url_without_downloading(client, app_module, url):
    with patch.object(app_module, "download") as download:
        response = client.post("/download", json={"url": url})
        assert response.status_code == 400
        download.assert_not_called()

def test_health(client):
    assert client.get("/health").json() == {"ok": True}

def test_invalid_body(client):
    assert client.post("/download", json={}).status_code == 422

def test_download_error_cleans_temp_folder(client, app_module, tmp_path, monkeypatch):
    created = tmp_path / "yt-test"
    def fake_mkdtemp(**kwargs):
        created.mkdir()
        return str(created)
    def fake_download(url, folder):
        (folder / "partial.mp4").write_bytes(b"partial")
        raise ValueError("download failed")
    monkeypatch.setattr(app_module.tempfile, "mkdtemp", fake_mkdtemp)
    monkeypatch.setattr(app_module, "download", fake_download)
    response = client.post("/download", json={"url": "https://youtu.be/test"})
    assert response.status_code == 422
    assert not created.exists()

def test_success_stream_and_cleanup(client, app_module, tmp_path, monkeypatch):
    created = tmp_path / "yt-success"
    def fake_mkdtemp(**kwargs):
        created.mkdir()
        return str(created)
    def fake_download(url, folder):
        result = folder / "video.mp4"
        result.write_bytes(b"fake-video")
        return result
    monkeypatch.setattr(app_module.tempfile, "mkdtemp", fake_mkdtemp)
    monkeypatch.setattr(app_module, "download", fake_download)
    response = client.post("/download", json={"url": "https://youtu.be/test"})
    assert response.status_code == 200
    assert response.content == b"fake-video"
    assert not created.exists()

def test_security_requires_service_token(app_module):
    with TestClient(app_module.app) as client:
        response = client.post(
            "/download",
            json={"url": "https://youtu.be/test"}
        )

    assert response.status_code == 403


def test_security_rejects_invalid_token(app_module):
    with TestClient(app_module.app) as client:
        response = client.post(
            "/download",
            json={"url": "https://youtu.be/test"},
            headers={
                "X-Service-Token": "invalid-token"
            }
        )

    assert response.status_code == 403

def test_show_routes(app_module):
    for route in app_module.app.routes:
        print(
            getattr(route, "path", None),
            getattr(route, "methods", None),
        )

def test_security_rejects_invalid_token(app_module):
    with TestClient(app_module.app) as client:
        response = client.post(
            "/download",
            json={"url": "https://youtu.be/test"},
            headers={
                "X-Service-Token": "wrong-token"
            }
        )

    assert response.status_code == 403