# CLI Commands

| Command                                    | Description                                                      |
| ------------------------------------------ | ---------------------------------------------------------------- |
| `pvmlab setup`                             | Installs dependencies and creates the artifacts directory.       |
| `pvmlab clean`                             | Stops all VMs and services, and removes the artifacts directory. |
| `pvmlab socket_vmnet start`                | Starts the `socket_vmnet` background service.                    |
| `pvmlab socket_vmnet stop`                 | Stops the `socket_vmnet` service.                                |
| `pvmlab socket_vmnet status`               | Checks the status of the `socket_vmnet` service.                 |
| `pvmlab vm create <name>`                  | Creates a new VM. Requires `--role`.                             |
| `pvmlab vm start <name>`                   | Starts the specified VM.                                         |
| `pvmlab vm stop <name>`                    | Stops the specified VM.                                          |
| `pvmlab vm shell <name>`                   | Opens an SSH session to the specified VM.                        |
| `pvmlab vm logs <name>`                    | Tails the console logs for the specified VM.                     |
| `pvmlab vm list`                           | Lists all VMs and their status.                                  |
| `pvmlab vm clean <name>`                   | Stops the VM and deletes its generated files.                    |
| `pvmlab vm docker start <vm> <tar>`        | Starts a docker container inside a VM.                           |
| `pvmlab vm docker stop <vm> <container>`   | Stops a docker container inside a VM.                            |
| `pvmlab vm docker status <vm>`             | Checks the status of docker containers inside a VM.              |
