#!/usr/bin/env python3
"""Fill a dev Cork instance with fake boards and pins.

Usage: python3 scripts/seed.py [--user NAME] [--data ./data] [--url http://localhost:8080]

It signs in as an existing user by adding a temporary session straight into
the SQLite database, so you don't need the password. Then it creates content
through the normal API and removes the session at the end. Images come from
picsum.photos, and each pin links to the photo's Unsplash page. A second user,
"demo" (password: demo-password), gets one public board.

For development only: never point this at a real instance.
"""

import argparse
import hashlib
import json
import random
import secrets
import sqlite3
import sys
import time
import urllib.request
import uuid
from pathlib import Path

# boards whose pins get a product price, like things saved from shops
PRICED = {"Desk setup", "Dream cabin"}
PRICES = ["24.99", "49.00", "129.00", "89.95", "349.00", "15.50", "1199.00", "64.00", "210.00"]

BOARDS = [
    ("Dream cabin", "Wood, stone, big windows. Somewhere quiet.", False),
    ("Travel bucket list", "Places to see before they're overrun.", True),
    ("Wallpapers", "Desktop and phone backgrounds.", False),
    ("Moody photography", "Low light, fog, long exposures.", False),
    ("City walks", "Streets, bridges and corners worth photographing.", True),
    ("Desk setup", "", False),
]

TITLES = [
    "Golden hour", "Morning fog", "Quiet corner", "Look at this light", "Saved for later",
    "Colour palette idea", "Weekend trip?", "Texture reference", "Composition study",
    "Love the tones here", "Next to-do", "Mood board", "Inspiration", "So calm",
    "Framing reference", "Rainy day", "Late night", "Off-season", "Overlook", "Simple and clean",
]

DESCRIPTIONS = [
    "",
    "",
    "Photo by {author}.",
    "Photo by {author}. The warm/cool contrast is what I'm after.",
    "Would make a great print for the hallway.",
    "Note to self: shoot at f/8, slightly underexposed.\nTry this again in autumn.",
    "Reference for the colour grade: crushed blacks, lifted greens.",
    "Found via a friend. Photo by {author}.",
]

DIMENSIONS = [(600, 400), (600, 600), (600, 750), (600, 800), (600, 900), (600, 1000), (600, 450)]


class Client:
    def __init__(self, base, token):
        self.base = base.rstrip("/")
        self.cookie = f"cork_session={token}"

    def call(self, method, path, body=None, form=None):
        headers = {"X-Cork": "1", "Cookie": self.cookie}
        data = None
        if form is not None:
            boundary = uuid.uuid4().hex
            headers["Content-Type"] = f"multipart/form-data; boundary={boundary}"
            parts = [
                f'--{boundary}\r\nContent-Disposition: form-data; name="{k}"\r\n\r\n{v}\r\n'
                for k, v in form.items()
            ]
            data = ("".join(parts) + f"--{boundary}--\r\n").encode()
        elif body is not None:
            headers["Content-Type"] = "application/json"
            data = json.dumps(body).encode()
        req = urllib.request.Request(self.base + path, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=60) as r:
                return json.load(r)
        except urllib.error.HTTPError as e:
            raise RuntimeError(f"{method} {path}: {e.code} {e.read().decode()}") from None


def temp_session(db, username):
    row = db.execute("SELECT id FROM users WHERE username = ? COLLATE NOCASE", (username,)).fetchone()
    if not row:
        sys.exit(f"no user named {username!r}; create it in the UI first")
    token = secrets.token_urlsafe(32)
    now = int(time.time() * 1000)
    db.execute(
        "INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)",
        (hashlib.sha256(token.encode()).hexdigest(), row[0], now, now + 3600_000),
    )
    db.commit()
    return token


def seed_boards(client, photos, boards, pins_per_board):
    created = []
    for name, desc, public in boards:
        board = client.call("POST", "/api/boards", {"name": name, "description": desc, "is_public": public})
        print(f"  board: {name}")
        for _ in range(random.randint(*pins_per_board)):
            photo = photos.pop()
            w, h = random.choice(DIMENSIONS)
            form = {
                "board_id": board["id"],
                "title": random.choice(TITLES) if random.random() < 0.8 else "",
                "description": random.choice(DESCRIPTIONS).format(author=photo["author"]),
                "source_url": photo["url"],
                "image_url": f"https://picsum.photos/id/{photo['id']}/{w}/{h}",
            }
            if name in PRICED and random.random() < 0.75:
                form["price"] = random.choice(PRICES)
                form["currency"] = random.choice(["USD", "USD", "USD", "EUR", "GBP"])
            try:
                pin = client.call("POST", "/api/pins", form=form)
                created.append(pin["id"])
                print(".", end="", flush=True)
            except RuntimeError as e:
                print(f"\n  skipped a pin: {e}")
        print()
    return created


def backdate(db, pin_ids, days=180):
    """Spread 'date added' over the past few months so the dates look realistic."""
    now = int(time.time() * 1000)
    for pid in pin_ids:
        ts = now - random.randint(0, days * 86400_000)
        db.execute("UPDATE pins SET created_at = ?, updated_at = ? WHERE id = ?", (ts, ts, pid))
    db.commit()


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--user", help="existing user to seed (default: the first admin)")
    ap.add_argument("--data", default="./data", help="Cork data directory")
    ap.add_argument("--url", default="http://localhost:8080", help="running Cork server")
    args = ap.parse_args()

    db_path = Path(args.data) / "cork.db"
    if not db_path.exists():
        sys.exit(f"{db_path} not found; start Cork and create an account first")
    db = sqlite3.connect(db_path)
    user = args.user or db.execute("SELECT username FROM users WHERE is_admin = 1 ORDER BY created_at").fetchone()[0]

    with urllib.request.urlopen("https://picsum.photos/v2/list?page=2&limit=100", timeout=30) as r:
        photos = json.load(r)
    random.shuffle(photos)

    token = temp_session(db, user)
    try:
        me = Client(args.url, token)
        print(f"Seeding {user}…")
        pins = seed_boards(me, photos, BOARDS, (6, 14))

        # a second user with a public board, for multi-user / public-view testing
        try:
            me.call("POST", "/api/admin/users", {"username": "demo", "password": "demo-password", "is_admin": False})
            demo_token = temp_session(db, "demo")
            try:
                print("Seeding demo…")
                pins += seed_boards(Client(args.url, demo_token), photos,
                                    [("Shared: Coastlines", "A public board from the demo user.", True)], (8, 10))
            finally:
                db.execute("DELETE FROM sessions WHERE token_hash = ?", (hashlib.sha256(demo_token.encode()).hexdigest(),))
                db.commit()
        except RuntimeError as e:
            print(f"Skipping demo user: {e}")

        backdate(db, pins)
        print(f"Done: {len(pins)} pins.")
    finally:
        db.execute("DELETE FROM sessions WHERE token_hash = ?", (hashlib.sha256(token.encode()).hexdigest(),))
        db.commit()


if __name__ == "__main__":
    main()
