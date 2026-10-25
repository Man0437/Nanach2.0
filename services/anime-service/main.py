
import os
import httpx
import random
from fastapi import FastAPI, HTTPException, Query

app = FastAPI(title="Anime Search Service")

API_URL = "https://api.yani.tv"
API_TOKEN = os.environ["YUMMY_API_TOKEN"]

# Проверь фактический адрес страницы на сайте.
SITE_URL = "https://yummyanime.tv"


def normalize_poster(poster: str | None):
    if not poster:
        return None

    if poster.startswith("//"):
        return "https:" + poster

    if poster.startswith("https://"):
        return poster

    return None


def normalize_anime(item: dict):
    rating = item.get("rating") or {}
    poster = item.get("poster") or {}
    status = item.get("anime_status") or {}

    slug = item.get("anime_url", "")

    return {
        "id": item.get("anime_id"),
        "title": item.get("title", "Без названия"),
        "year": item.get("year") or None,
        "poster": normalize_poster(
            poster.get("medium")
        ),
        "url": f"{SITE_URL}/{slug}" if slug else None,
        "rating": round(
            rating.get("average") or 0, 2
        ),
        "status": status.get("title", "Неизвестен")
    }


@app.get("/search")
async def search_anime(
    q: str = Query(min_length=1, max_length=150)
):
    headers = {
        "X-Application": API_TOKEN,
        "Lang": "ru",
        "Accept": "application/json"
    }

    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            response = await client.get(
                f"{API_URL}/search",
                params={"q": q},
                headers=headers
            )

            response.raise_for_status()
            data = response.json()

    except httpx.TimeoutException:
        raise HTTPException(504, "API timeout")

    except httpx.HTTPError:
        raise HTTPException(502, "YummyAnime API unavailable")

    anime_list = data.get("response", [])

    return {
        "count": len(anime_list),
        "results": [
            normalize_anime(item)
            for item in anime_list[:5]
        ]
    }


@app.get("/health")
def health():
    return {"status": "ok"}

RANDOM_QUERIES = (
    "аниме",
    "приключения",
    "фантастика",
    "романтика",
    "комедия",
    "драма",
    "фэнтези",
)


@app.get("/random")
async def random_anime():
    query = random.choice(RANDOM_QUERIES)

    search_result = await search_anime(q=query)
    results = search_result["results"]

    if not results:
        raise HTTPException(
            status_code=404,
            detail="Не удалось найти случайное аниме"
        )

    return {
        "result": random.choice(results)
    }