import socketserver

# Persistent global namespace for exec/eval
global_namespace = {}

class PythonExecHandler(socketserver.StreamRequestHandler):
    def handle(self):
        code = self.rfile.readline().strip().decode("utf-8")
        try:
            exec(code, global_namespace)
            self.wfile.write(b"OK\n")
        except Exception as e:
            self.wfile.write(f"ERROR: {e}\n".encode())

if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="Persistent Python Exec Server")
    parser.add_argument("--port", type=int, default=9000, help="TCP port to listen on")
    args = parser.parse_args()

    server = socketserver.TCPServer(("0.0.0.0", args.port), PythonExecHandler)
    print(f"Python exec server running on port {args.port}")
    server.serve_forever()
