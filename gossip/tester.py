import subprocess
import time


def run_command(command):
    try:
        subprocess.Popen(command, shell=True)
        # process = subprocess.Popen(
        #     command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, shell=True)

        # # Wait for the process to complete and capture the output and error
        # stdout, stderr = process.communicate()

        # if process.returncode != 0:
        #     print(f"Command failed with return code {process.returncode}")
        #     return (stderr)
        # else:
        #     return (stdout)

    except Exception as e:
        print(f"An error occurred: {e}")


nodes = ["127.0.0.1", "127.0.0.2", "127.0.0.3", "127.0.0.4", "127.0.0.5"]

edges = [
    (nodes[1], nodes[0]),
    (nodes[2], nodes[0]),
    (nodes[3], nodes[1]),
    (nodes[4], nodes[2]),
    (nodes[4], nodes[3]),
]

# For now cache size and degree is equal for everybody
cache_size = 50
degree = 30

full_commands = []

for node in nodes:
    vaddr = node + ":6001"
    # Horizontal port is 7001
    haddr = node + ":7001"
    peers = ""
    for edge in edges:
        if edge[0] == node:
            peers += edge[1] + ":7001 "

    command = f"./gossip --degree {degree} --cache {cache_size} --vaddr {vaddr} --haddr {haddr}  {peers}"
    full_commands.append(command)


subprocess.run(["make", "build"])
for command in full_commands:
    print(command)
    time.sleep(.1)
    run_command(command)

# OUTPUT not really readable, loggin level should be changed
