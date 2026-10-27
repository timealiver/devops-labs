import os
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse, parse_qs

hoard = []


def burn():
    while True:
        pass


class Handler(BaseHTTPRequestHandler):
    def reply(self, body, code=200):
        data = body.encode()
        self.send_response(code)
        self.send_header("Content-Type", "text/plain")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        url = urlparse(self.path)
        if url.path == "/health":
            self.reply("ok")
        elif url.path == "/eat":
            mb = int(parse_qs(url.query).get("mb", ["0"])[0])
            chunk = bytearray(mb * 1024 * 1024)
            for i in range(0, len(chunk), 4096):
                chunk[i] = 1
            hoard.append(chunk)
            self.reply(f"holding {sum(len(c) for c in hoard) // 1024 // 1024} MB\n")
        elif url.path == "/burn":
            threading.Thread(target=burn, daemon=True).start()
            self.reply("burning one core\n")
        else:
            self.reply("not found\n", 404)

    def log_message(self, fmt, *args):
        pass


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8080"))
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()
