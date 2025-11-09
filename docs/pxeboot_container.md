# Interacting with the pxeboot Container

The provisioner VM runs a Docker container that includes the entire pxeboot stack (DHCP, TFTP, HTTP services). This container is built from the sources in the `pxeboot_stack/` directory.

While you can interact with Docker directly inside the provisioner VM, the `pvmlab` CLI provides convenient wrapper commands to manage the container's lifecycle.

## Managing the Container with `pvmlab`

These commands are the recommended way to manage the pxeboot stack.

### Starting and Updating the Container

The `pvmlab provisioner docker start` command is used to both initially start the container and to update it with a new image.

```bash
pvmlab provisioner docker start <path/to/pxeboot_stack.tar> --network-host --privileged
```

This command performs the following steps:

1. Stops and removes any existing `pxeboot_stack` container.
2. Loads the new Docker image from the specified `.tar` file.
3. Starts a new container named `pxeboot_stack` with the specified options.

The `--network-host` and `--privileged` flags are required for the container to correctly manage network services for the lab.

### Stopping the Container

To stop the pxeboot stack without removing the container:

```bash
pvmlab provisioner docker stop
```

### Checking the Container Status

To check if the container is running and see its status:

```bash
pvmlab provisioner docker status
```

## Development Workflow

If you make changes to the `pxeboot_stack` source code (e.g., `Dockerfile`, `supervisord.conf`), you can update the running container without recreating the entire provisioner VM.

1. **On your Mac**, rebuild the Docker image and create the tarball:

    ```bash
    make -C pxeboot_stack tar
    ```

    This command builds the image and creates `pxeboot_stack.tar` in the `pxeboot_stack/` directory.

2. **In another terminal**, update the container in the provisioner VM using the `start` command. This will replace the old container with the new one.

    ```bash
    pvmlab provisioner docker start ./pxeboot_stack/pxeboot_stack.tar --network-host --privileged
    ```

3. **Check the status** to ensure the new container is running correctly:

    ```bash
    pvmlab provisioner docker status
    ```

This workflow allows for a rapid development cycle when working on the provisioning services.

## Direct Interaction with Docker

For more advanced debugging, you can get a shell inside the provisioner VM and use standard `docker` commands.

1. **Get a shell inside the provisioner VM:**

    ```bash
    pvmlab vm shell provisioner
    ```

2. **Interact with the container:**

    Once inside the VM, you can use any `docker` command:

    ```bash
    # Get an interactive shell inside the running container
    docker exec -it pxeboot_stack /bin/sh

    # View the container's logs in real-time
    docker logs -f pxeboot_stack

    # Inspect the container's configuration
    docker inspect pxeboot_stack
    ```
