# docker-compose-exec

A CLI tool for discovering Docker Compose files and services across multiple directories. It provides an interactive terminal UI for selecting Compose files and services, then executes custom commands (e.g., docker exec) on the selected service. 

## Features  

- Automatically searches for Docker Compose files (`docker-compose.yml`, `compose.yml`, etc.) in predefined paths.  
- Interactive terminal-based UI for selecting Compose files and services.  
- Executes a configurable shell command on the selected service.  

## Installation  

1. Download the latest release binary from the [releases page](https://github.com/your-username/docker-compose-exec/releases).  
2. Copy the binary to `/usr/local/bin`:  
```bash
sudo mv docker-compose-exec /usr/local/bin
```
3. Make the binary executable:
```bash
sudo chmod +x /usr/local/bin/docker-compose-exec
```

## Usage

Just run:
```bash
docker-compose-exec
```

### Optional Environment Variables

You can configure the following environment variables:

- CONTAINER_BASE_PATH
    Specify paths to search for Docker Compose files (colon-separated). Example:
    ```bash
    export CONTAINER_BASE_PATH="/path/to/containers:/another/path"
    ```
    Default paths include `/var/container` and `/srv/container`.
- CONTAINER_EXEC_COMMAND
    Customize the execution command. Example:
    ```bash
    export CONTAINER_EXEC_COMMAND="docker compose -f %COMPOSE exec --user root %SERVICE /bin/bash"
    ```
    - `%COMPOSE` will be replaced with the path to the selected Compose file.
    - `%SERVICE` will be replaced with the selected service.

## Example

Imagine you have the following directory structure:

```bash
/var/container  
├── project1/  
│   └── docker-compose.yml  
├── project2/  
│   └── compose.yml  
/srv/container  
└── project3/  
    └── docker-compose.yaml  
```

Running `docker-compose-exec` will:
- Discover these Compose files.
- Allow you to select a file (e.g., project1/docker-compose.yml).
- List available services from the selected file.
- Execute the configured command (e.g., docker exec) on the chosen service.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request with any enhancements or bug fixes.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
