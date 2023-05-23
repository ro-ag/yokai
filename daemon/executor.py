import zmq
import subprocess
import os
context = zmq.Context()
socket = context.socket(zmq.REP)
socket.bind("ipc:///tmp/daemon.ipc")  # IPC socket
s_pid = str(os.getpid())
print("running daemon from pid: " + s_pid)

while True:
    #  Wait for next request from client
    script_path = socket.recv_string()
    print("Received request to run: ", script_path)

    # Execute the script and get the return code
    process = subprocess.run(script_path, shell=True)
    return_code = process.returncode

    #  Send reply back to client
    socket.send_string(str(return_code))
