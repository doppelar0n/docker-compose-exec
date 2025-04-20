Usage: docker-compose-exec [OPTIONS]

A CLI tool for discovering Docker Compose files and their services across multiple directories.  
Select a Compose file and service interactively, then execute a custom command on the service.

Options:
  --help                Show this help message and exit
  --version             Show the tool's version and exit

Environment Variables:
  CONTAINER_BASE_PATH                Additional paths to search for Compose files, separated by ":"
  CONTAINER_BASE_PATH_MAX_DEPTH      Compose file max search depth (Default: 2)
  CONTAINER_EXEC_COMMAND             Default command template for execution (e.g., "docker compose -f %COMPOSE exec %SERVICE /bin/bash")
  CONTAINER_EXEC_COMMAND_NOT_RUNNING Command template for execution if docker container is not running

Examples:
  docker-compose-exec
      Launch the tool with default paths and interactive UI.

  CONTAINER_BASE_PATH="/var/mycontainers:/srv/containers" docker-compose-exec
      Specify custom search paths for Compose files.

  CONTAINER_EXEC_COMMAND="docker compose -f %COMPOSE exec %SERVICE /bin/bash" docker-compose-exec
      Use a custom execution command.

  CONTAINER_EXEC_COMMAND="echo %COMPOSE %SERVICE" docker-compose-exec
      This is like dry run.

Contributions & Support:
  Bugs, feature requests, improvements, or contributions are welcome!  
  For updates and more information, visit:  
  https://github.com/doppelar0n/docker-compose-exec
