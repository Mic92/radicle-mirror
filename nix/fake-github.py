import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

APP_ID = int(os.environ["APP_ID"])
INSTALLATION_ID = int(os.environ["INSTALLATION_ID"])
OWNER_LOGIN = os.environ["OWNER_LOGIN"]
OWNER_ID = int(os.environ["OWNER_ID"])
REPO_ID = int(os.environ["REPO_ID"])
REPO_NAME = os.environ["REPO_NAME"]
REPO_FULLNAME = os.environ["REPO_FULLNAME"]
CLONE_URL = os.environ["CLONE_URL"]
RID_FILE = os.environ["RID_FILE"]
CHECK_RUN_FILE = os.environ["CHECK_RUN_FILE"]

REPO = {
    "id": REPO_ID,
    "name": REPO_NAME,
    "full_name": REPO_FULLNAME,
    "pushed_at": "2026-06-21T06:07:48Z",
    "description": "integration test repo",
    "private": False,
    "owner": {"login": OWNER_LOGIN, "id": OWNER_ID},
    "clone_url": CLONE_URL,
}


def load_rid():
    try:
        with open(RID_FILE) as f:
            return f.read().strip()
    except FileNotFoundError:
        return ""


class Handler(BaseHTTPRequestHandler):
    def _json(self, status, body):
        payload = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self):
        path = self.path.split("?", 1)[0]
        if path == "/app/installations":
            self._json(
                200,
                [
                    {
                        "id": INSTALLATION_ID,
                        "app_id": APP_ID,
                        "account": {"login": OWNER_LOGIN},
                    }
                ],
            )
        elif path == "/installation/repositories/":
            self._json(200, {"repositories": [REPO], "total_count": 1})
        elif path.endswith("/actions/variables/RADICLE_RID"):
            rid = load_rid()
            if rid:
                self._json(200, {"name": "RADICLE_RID", "value": rid})
            else:
                self._json(404, {"message": "Not Found"})
        else:
            self._json(404, {"message": "Not Found"})

    def do_POST(self):
        if self.path.endswith("/access_tokens"):
            self._json(201, {"token": "faketoken"})
        elif self.path.endswith("/check-runs"):
            length = int(self.headers.get("Content-Length", "0"))
            body = json.loads(self.rfile.read(length) or "{}")
            with open(CHECK_RUN_FILE, "w") as f:
                json.dump(body, f)
            self._json(201, {"id": 1})
        else:
            self._json(404, {"message": "Not Found"})

    def do_PATCH(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length) or "{}")
        if self.path.endswith("/actions/variables/RADICLE_RID"):
            with open(RID_FILE, "w") as f:
                f.write(body.get("value", ""))
            self._json(204, {})
        else:
            self._json(404, {"message": "Not Found"})

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", 3000), Handler).serve_forever()
