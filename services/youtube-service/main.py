import asyncio
import os
import shutil
import tempfile
from pathlib import Path
from urllib.parse import urlparse
import secrets

import yt_dlp
from fastapi import FastAPI, HTTPException, Depends, Header
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

app = FastAPI(title="YouTube Video Service")
MAX_BYTES = int(os.getenv("MAX_VIDEO_BYTES", str(45 * 1024 * 1024)))
MAX_DURATION = int(os.getenv("MAX_DURATION_SECONDS", "900"))
MAX_CONCURRENT = int(os.getenv("MAX_CONCURRENT_DOWNLOADS", "2"))
sem = asyncio.Semaphore(MAX_CONCURRENT)
SERVICE_TOKEN = os.environ["YOUTUBE_SERVICE_TOKEN"]

def verify_service_token(
    x_service_token: str = Header(default="")
):
    if not secrets.compare_digest(
        x_service_token,
        SERVICE_TOKEN
    ):
        raise HTTPException(
            status_code=403,
            detail="Forbidden"
        )

class DownloadRequest(BaseModel):
    url: str

def valid_url(value: str) -> bool:
    try:
        parsed = urlparse(value)
        host = (parsed.hostname or "").lower()
        return parsed.scheme == "https" and host in {
            "youtube.com", "www.youtube.com", "m.youtube.com", "youtu.be"
        } and not parsed.username and not parsed.password and not parsed.port
    except ValueError:
        return False

def download(url: str, folder: Path) -> Path:
    opts = {
        "outtmpl": str(folder / "video.%(ext)s"),
        "format": "bestvideo[height<=480][ext=mp4]+bestaudio[ext=m4a]/best[height<=480][ext=mp4]/best[height<=480]",
        "merge_output_format": "mp4",
        "noplaylist": True,
        "restrictfilenames": True,
        "quiet": True,
        "no_warnings": True,
        "cachedir": False,
        "socket_timeout": 20,
        "max_filesize": MAX_BYTES,
        "retries": 2,
        "extractor_args": {"youtube": {"player_client": ["android", "web"]}},
    }
    with yt_dlp.YoutubeDL(opts) as ydl:
        info = ydl.extract_info(url, download=False)
        if not info or info.get("_type") == "playlist":
            raise ValueError("Укажите ссылку на одно видео")
        if info.get("duration") and info["duration"] > MAX_DURATION:
            raise ValueError(f"Видео длиннее {MAX_DURATION} секунд")
        ydl.download([url])
    files = list(folder.glob("video.mp4"))
    if not files:
        raise ValueError("Не удалось получить MP4 (проверьте FFmpeg и форматы)")
    file = files[0]
    if file.stat().st_size > MAX_BYTES:
        raise ValueError(f"Файл превышает лимит {MAX_BYTES // 1048576} МиБ")
    return file

@app.get("/health")
def health():
    return {"ok": True}

@app.post(
    "/download",
    dependencies=[Depends(verify_service_token)]
)
async def download_video(request: DownloadRequest):
    if not valid_url(request.url):
        raise HTTPException(400, "Нужна HTTPS-ссылка на youtube.com или youtu.be")
    if sem.locked():
        raise HTTPException(429, "Сервис занят, повторите позже")
    async with sem:
        folder = Path(tempfile.mkdtemp(prefix="yt-", dir="/tmp"))
        try:
            file = await asyncio.to_thread(download, request.url, folder)
        except Exception as exc:
            shutil.rmtree(folder, ignore_errors=True)
            raise HTTPException(422, f"Ошибка скачивания: {exc}") from exc

        def chunks():
            try:
                with file.open("rb") as handle:
                    while chunk := handle.read(1024 * 1024):
                        yield chunk
            finally:
                shutil.rmtree(folder, ignore_errors=True)

        return StreamingResponse(
            chunks(), media_type="video/mp4",
            headers={
                "Content-Disposition": 'attachment; filename="video.mp4"',
                "Content-Length": str(file.stat().st_size),
            },
        )
