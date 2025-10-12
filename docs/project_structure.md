# Project Structure

The project is a Go CLI application with the following structure:

```shell
.
├── internal/               # Internal packages not intended for reuse
├── pvmlab/                 # Main application package
│   ├── cmd/                # Cobra command definitions
│   └── main.go
├── pxeboot_stack/          # Source for the Docker container running the pxeboot stack
├── go.mod                  # Go module definition
├── go.sum                  # Go module dependencies
└── README.md
```
