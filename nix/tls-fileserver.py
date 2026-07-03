import http.server
import os
import ssl
import sys

root, cert, key, port = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
os.chdir(root)
httpd = http.server.HTTPServer(("127.0.0.1", port), http.server.SimpleHTTPRequestHandler)
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain(cert, key)
httpd.socket = ctx.wrap_socket(httpd.socket, server_side=True)
httpd.serve_forever()
