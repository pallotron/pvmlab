# Interacting with the pxeboot Container

The provisioner VM has a Docker container running the pxeboot stack.
The container is defined in `pxeboot_stack/Dockerfile` but you can also provide your own (in `.tar` format).
Some commands are defined to manage the container, see `pvmlab vm docker --help`, but you can also use `docker` commands directly
once you are inside the container.

**Get a Shell Inside the Container:**

Get into the provisioner:

```bash
pvmlab vm shell provisioner
```

For debugging, you can get an interactive shell.

```bash
docker exec -it pxeboot_stack /bin/sh
```

**Manage Container Lifecycle:**

Standard `docker` commands apply here:

```bash
# Stop the container
docker stop pxeboot_stack
# Start the container
docker start pxeboot_stack
# Restart the container
docker restart pxeboot_stack

# Cleanup; first, remove any stopped containers that may reference old images
docker container prune
# Then, remove dangling images
docker image prune

# tail files inside the container
docker exec pxeboot_stack tail -f /path/to/file.log
```

## Building and Reloading the pxeboot Docker Container

The `pxeboot_stack` Docker image is built on your macOS host and then loaded into the provisioner VM.

**Initial Setup:**

When you first create the provisioner VM (`pvmlab vm create provisioner`), the following happens automatically via `cloud-init`:

1. Docker is installed in the VM.
2. The `~/.provisioning-vm-lab/docker_images/` directory from your Mac is mounted into the VM at `/mnt/host/docker_images/`.
3. The `pxeboot_stack.tar` image is loaded into Docker.
4. The `pxeboot_stack` container is started.

**Development Workflow:**

If you make changes to the `pxeboot_stack` source code (e.g., `Dockerfile`, `supervisord.conf`), you can update the running container without recreating the entire VM:

1. **On your Mac**, rebuild and save the Docker image:

   ```bash
   make -C pxeboot_stack tar
   ```

   This command builds the image and creates the `pxeboot_stack.tar` file in the `pxeboot_stack` directory inside the git repo.

2. **Update the container in the provisioner VM**:

   ```bash
   pvmlab vm docker start provisioner ./pxeboot_stack/pxeboot_stack.tar --network-host --privileged
   ```

   You can also provide a full path to your pxeboot stack tarball if you are developing your own outside of the project.

3. **Check the container status**:
   This script stops and removes the old container, loads the updated image from the shared directory, and starts a new container.

   ```bash
   pvmlab vm docker status provisioner
   ```

This allows for a rapid development cycle when working on the provisioning services.
