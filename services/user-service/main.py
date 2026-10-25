from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()
users = {}

class User(BaseModel):
    telegram_id: int
    username: str | None = None

@app.post("/users")
def create_user(user: User):
    users[user.telegram_id] = user.model_dump()
    return {"status": "create", "user": user}

@app.get("/users/{telegram_id}")
def get_user(telegram_id: int):
    if telegram_id not in users:
        return {"error": "User not found"}

    return users[telegram_id]

@app.get("/health")
def health():
    return {"status": "ok"}