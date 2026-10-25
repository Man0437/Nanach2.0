import os, sqlite3, secrets, time, json
from contextlib import contextmanager
from datetime import datetime, timezone, timedelta
from fastapi import FastAPI, Request, HTTPException, Depends
from fastapi.responses import HTMLResponse, JSONResponse
from fastapi.security import HTTPBasic, HTTPBasicCredentials
from pydantic import BaseModel, Field

DB=os.getenv('DB_PATH','/data/admin.db')
ADMIN_USER=os.environ['ADMIN_USER']; ADMIN_PASSWORD=os.environ['ADMIN_PASSWORD']; EVENT_TOKEN=os.environ['EVENT_TOKEN']
app=FastAPI(docs_url=None,redoc_url=None,openapi_url=None)
basic=HTTPBasic()
@contextmanager
def db():
    c=sqlite3.connect(DB,timeout=10)
    c.row_factory=sqlite3.Row
    try:
        yield c
        c.commit()
    finally: c.close()
@app.on_event('startup')
def init():
    with db() as c:
        c.execute('PRAGMA journal_mode=WAL')
        c.execute('CREATE TABLE IF NOT EXISTS users (telegram_id INTEGER PRIMARY KEY, username TEXT, first_seen TEXT NOT NULL, last_seen TEXT NOT NULL, searches INTEGER NOT NULL DEFAULT 0)')
        c.execute('CREATE TABLE IF NOT EXISTS logs (id INTEGER PRIMARY KEY AUTOINCREMENT, ts TEXT NOT NULL, level TEXT NOT NULL, service TEXT NOT NULL, event TEXT NOT NULL, telegram_id INTEGER, detail TEXT)')
        c.execute('CREATE INDEX IF NOT EXISTS logs_ts ON logs(id DESC)')
        c.execute('CREATE INDEX IF NOT EXISTS logs_activity_ts ON logs(ts, telegram_id)')
        c.execute('CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)')
        c.execute("INSERT OR IGNORE INTO settings(key,value) VALUES ('refresh_seconds','10'),('activity_days','7')")
def admin_auth(c:HTTPBasicCredentials=Depends(basic)):
    if not (secrets.compare_digest(c.username.encode(),ADMIN_USER.encode()) and secrets.compare_digest(c.password.encode(),ADMIN_PASSWORD.encode())):
        raise HTTPException(401,'Unauthorized',headers={'WWW-Authenticate':'Basic realm="Telegram Admin"'})
def event_auth(r:Request):
    token=r.headers.get('X-Event-Token','')
    if not secrets.compare_digest(token.encode(),EVENT_TOKEN.encode()): raise HTTPException(403,'Forbidden')
class Event(BaseModel):
    level:str='INFO'
    service:str='bot-service'
    event:str=Field(min_length=1,max_length=100)
    telegram_id:int|None=None
    username:str|None=None
    detail:str|None=None
@app.post('/internal/events',dependencies=[Depends(event_auth)])
def event(item:Event):
    now=datetime.now(timezone.utc).isoformat(timespec='seconds')
    with db() as c:
        if item.telegram_id is not None:
            c.execute('INSERT INTO users(telegram_id,username,first_seen,last_seen,searches) VALUES(?,?,?,?,?) ON CONFLICT(telegram_id) DO UPDATE SET username=COALESCE(excluded.username,users.username),last_seen=excluded.last_seen,searches=users.searches+excluded.searches',(item.telegram_id,item.username,now,now,int(item.event=='anime_search')))
        c.execute('INSERT INTO logs(ts,level,service,event,telegram_id,detail) VALUES(?,?,?,?,?,?)',(now,item.level[:16],item.service[:60],item.event,item.telegram_id,(item.detail or '')[:2000]))
        c.execute('DELETE FROM logs WHERE id NOT IN (SELECT id FROM logs ORDER BY id DESC LIMIT 20000)')
    return {'ok':True}
@app.get('/api/overview',dependencies=[Depends(admin_auth)])
def overview():
    with db() as c:
        users=c.execute('SELECT COUNT(*) FROM users').fetchone()[0]
        searches=c.execute("SELECT COUNT(*) FROM logs WHERE event='anime_search' AND ts>=strftime('%Y-%m-%dT%H:%M:%S','now','-1 day')").fetchone()[0]
        errors=c.execute("SELECT COUNT(*) FROM logs WHERE level='ERROR' AND ts>=strftime('%Y-%m-%dT%H:%M:%S','now','-1 day')").fetchone()[0]
        return {'users':users,'searches_24h':searches,'errors_24h':errors}
@app.get('/api/users',dependencies=[Depends(admin_auth)])
def users():
    with db() as c: return [dict(x) for x in c.execute('SELECT * FROM users ORDER BY last_seen DESC LIMIT 200')]
@app.get('/api/logs',dependencies=[Depends(admin_auth)])
def logs():
    with db() as c: return [dict(x) for x in c.execute('SELECT * FROM logs ORDER BY id DESC LIMIT 200')]
@app.get('/',response_class=HTMLResponse,dependencies=[Depends(admin_auth)])
def home():
    with open(os.path.join(os.path.dirname(__file__), 'static', 'index.html'),encoding='utf-8') as f:return f.read()


class SettingsUpdate(BaseModel):
    refresh_seconds: int = Field(ge=5, le=300)
    activity_days: int = Field(ge=1, le=30)

@app.get('/api/settings', dependencies=[Depends(admin_auth)])
def get_settings():
    with db() as c:
        return {r['key']: int(r['value']) for r in c.execute(
            "SELECT key,value FROM settings WHERE key IN ('refresh_seconds','activity_days')"
        )}

@app.put('/api/settings', dependencies=[Depends(admin_auth)])
def put_settings(settings: SettingsUpdate):
    with db() as c:
        for key, value in settings.model_dump().items():
            c.execute('UPDATE settings SET value=? WHERE key=?', (str(value), key))
    return settings.model_dump()

@app.get('/api/activity', dependencies=[Depends(admin_auth)])
def activity(days: int = 7):
    if days < 1 or days > 30:
        raise HTTPException(422, 'days must be between 1 and 30')
    today = datetime.now(timezone.utc).date()
    start = today - timedelta(days=days - 1)
    with db() as c:
        rows = c.execute('''
            SELECT substr(ts,1,10) AS day,
                   COUNT(*) AS events,
                   COUNT(DISTINCT telegram_id) AS active_users
            FROM logs
            WHERE telegram_id IS NOT NULL AND ts >= ?
            GROUP BY substr(ts,1,10)
            ORDER BY day
        ''', (start.isoformat(),)).fetchall()
    by_day = {r['day']: dict(r) for r in rows}
    return {'days': [
        {'date': (start + timedelta(days=i)).isoformat(),
         'events': by_day.get((start + timedelta(days=i)).isoformat(), {}).get('events', 0),
         'active_users': by_day.get((start + timedelta(days=i)).isoformat(), {}).get('active_users', 0)}
        for i in range(days)
    ]}
