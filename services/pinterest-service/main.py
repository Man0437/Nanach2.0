import json
import os
import random
from pathlib import Path
from urllib.parse import urlparse

from fastapi import FastAPI, HTTPException

app = FastAPI(title="Pinterest Image Service")

CATALOG_PATH = Path(
    os.getenv("PINTEREST_CATALOG", "/app/images.json")
)


def valid_image_url(value: str) -> bool:
    try:
        parsed = urlparse(value)
        return (
            parsed.scheme == "https"
            and bool(parsed.hostname)
            and parsed.username is None
            and parsed.password is None
            and parsed.port is None
        )
    except ValueError:
        return False


@app.get("/health")
def health():
    return {"status": "ok"}


@app.get("/random")
def random_image():
    try:
        images = json.loads(
            CATALOG_PATH.read_text(encoding="utf-8")
        )
    except (OSError, json.JSONDecodeError):
        raise HTTPException(503, "Image catalog unavailable")

    if not isinstance(images, list):
        raise HTTPException(503, "Invalid image catalog")

    available = [
        item for item in images
        if isinstance(item, dict)
        and isinstance(item.get("image_url"), str)
        and valid_image_url(item["image_url"])
    ]

    if not available:
        raise HTTPException(404, "No images available")

    return {"result": random.choice(available)}