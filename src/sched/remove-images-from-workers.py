import subprocess as sp
import sys

def run_command(command):
    result = sp.run(command, stdout=sp.PIPE, stderr=sp.PIPE, shell=True, universal_newlines=True)
    return result.stdout.strip()

cluster_name = run_command("kind get clusters")

workers = run_command(f"docker ps | grep {cluster_name}-worker | awk '{{ print $1 }}'")

image_to_remove = sys.argv[1]

for container_id in workers.splitlines():
    run_command(f"docker exec {container_id} crictl rmi {image_to_remove}")

print(f'removed images for {len(workers.splitlines())} workers')
