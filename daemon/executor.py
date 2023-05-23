import zmq
import subprocess
import os
import psutil

# Create a ZeroMQ context
context = zmq.Context()

# Create a REP (reply) socket
socket = context.socket(zmq.REP)

# Bind the socket to an IPC endpoint
socket.bind("ipc:///tmp/daemon.ipc")

# Print the PID of this script for debugging purposes
print("running daemon from pid: ", os.getpid())

# Main loop
while True:
    # Wait for next request from client
    script_path = socket.recv_string()
    print("Received request to run: ", script_path)

    # Execute the script and get the return code
    try:
        process = subprocess.run(script_path, shell=True, timeout=5)
        return_code = process.returncode
    except subprocess.TimeoutExpired:
        print("Script took too long to run.")
        return_code = 1  # Indicate an error

    # Get memory usage
    process = psutil.Process(os.getpid())
    memory_info = process.memory_info()
    print("Memory Info:", memory_info)

    # Send reply back to client
    socket.send_string(str(return_code))
