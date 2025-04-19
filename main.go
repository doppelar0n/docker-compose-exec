package main

import (
	_ "embed"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var defaultContainerBasePath = "/var/container:/srv/container"
var defaultContainerExecCommand = "docker compose -f %COMPOSE exec --user root %SERVICE /bin/sh"
var defaultContainerExecCommandNotRunning = "docker compose -f %COMPOSE exec --user root %SERVICE /bin/sh"
var defaultMaxDepth = 2

//go:embed HELP.md
var help string

var version = "development"

func isDirectory(path string) (bool) {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func getComposeFilesInDir(basePath string, maxDepth int) ([]string, error) {
	isDir := isDirectory(basePath)
	if !isDir {
		return nil, fmt.Errorf("%s is not a directory", basePath)
	}

	composeFileNames := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	var composeFiles []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(basePath, path)
		
		if err != nil {
			return err
		}
		depth := strings.Count(relativePath, string(os.PathSeparator))
		if depth >= maxDepth {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			for _, fileName := range composeFileNames {
				if filepath.Base(path) == fileName {
					composeFiles = append(composeFiles, path)
					break
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error while scanning directory %s: %w", basePath, err)
	}

	if len(composeFiles) == 0 {
		return nil, fmt.Errorf("no docker compose files found in %s", basePath)
	}

	return composeFiles, nil
}

func getAllComposeSearchPaths() []string {
	var paths []string

	containerBasePathString := os.Getenv("CONTAINER_BASE_PATH")
	if containerBasePathString == "" {
		containerBasePathString = defaultContainerBasePath
	}

	containerBasePaths := strings.Split(containerBasePathString, ":")

	for _, path := range containerBasePaths {
		trimmedPath := strings.TrimSuffix(path, "/")
		if !slices.Contains(paths, trimmedPath) {
			paths = append(paths, trimmedPath)
		}
	}

	return paths
}

func getAllComposeFiles() ([]string, string, error) {
	var composeFilePaths []string
	paths := getAllComposeSearchPaths()

	maxDepthString := os.Getenv("CONTAINER_BASE_PATH_MAX_DEPTH")
	maxDepth, err := strconv.Atoi(maxDepthString)
	if err != nil {
		if maxDepthString != "" {
			fmt.Printf("Invalid value for CONTAINER_BASE_PATH_MAX_DEPTH: %s. Using default: %d\n", maxDepthString, defaultMaxDepth)
		}
		maxDepth = defaultMaxDepth
	}
	if maxDepth < 1 {
		maxDepth = 1
	}

	for _, path := range paths {
		currentComposeFilePaths, _ := getComposeFilesInDir(path, maxDepth)
		composeFilePaths = append(composeFilePaths, currentComposeFilePaths...)
	}

	return composeFilePaths, strings.Join(paths, " "), nil
}

func getDockerServiceArray(dockerComposeYaml string) ([]string, error) {
	var serviceKeys []string

	data, err := os.ReadFile(dockerComposeYaml)
	if err != nil {
		return nil, err
	}

	var rawData map[string]interface{}
	err = yaml.Unmarshal(data, &rawData)
	if err != nil {
		return nil, err
	}

	services, ok := rawData["services"].(map[string]interface{})
	if !ok {
		return nil, errors.New("rawData[\"services\"].(map[string]interface{}) is no ok")
	}

	for key := range services {
		serviceKeys = append(serviceKeys, key)
	}

	return serviceKeys, nil
}

func isDockerRunning(dockerComposeYaml string, dockerService string) bool {
	cmd := exec.Command("docker", fmt.Sprintf("compose -f %s ps %s --format json", dockerComposeYaml, dockerService))

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(out.Bytes(), &jsonData)
	if err != nil {
		return false
	}
	if len(jsonData) == 0 {
		return false
	}
	if jsonData["State"] == "running" {
		return true
	}

	return false
}

func runDockerExec(dockerComposeYaml string, dockerService string) error {
	dockerExecCommand := os.Getenv("CONTAINER_EXEC_COMMAND")
	if dockerExecCommand == "" {
		dockerExecCommand = defaultContainerExecCommand
	}

	dockerExecCommandNotRunningExec := os.Getenv("CONTAINER_EXEC_COMMAND_NOT_RUNNING")
	if dockerExecCommandNotRunningExec == "" {
		dockerExecCommandNotRunningExec = defaultContainerExecCommandNotRunning
	}

	dockerExecCommandParts := strings.Split(dockerExecCommand, " ")
	for i, part := range dockerExecCommandParts {
		switch part {
		case "%COMPOSE":
			dockerExecCommandParts[i] = dockerComposeYaml
		case "%SERVICE":
			dockerExecCommandParts[i] = dockerService
		}
	}

	dockerExecCommandNotRunningExecParts := strings.Split(dockerExecCommandNotRunningExec, " ")
	for i, part := range dockerExecCommandNotRunningExecParts {
		switch part {
		case "%COMPOSE":
			dockerExecCommandNotRunningExecParts[i] = dockerComposeYaml
		case "%SERVICE":
			dockerExecCommandNotRunningExecParts[i] = dockerService
		}
	}

	var cmd *exec.Cmd
	if isDockerRunning(dockerComposeYaml, dockerService) {
		fmt.Printf("Docker container %s %s is running\n", dockerComposeYaml, dockerService)
		fmt.Printf("exec:  %s\n", strings.Join(dockerExecCommandParts, " "))
		cmd = exec.Command(dockerExecCommandParts[0], dockerExecCommandParts[1:]...)
	} else {
		fmt.Printf("Docker container %s %s is NOT running\n", dockerComposeYaml, dockerService)
		fmt.Printf("exec:  %s\n", strings.Join(dockerExecCommandNotRunningExecParts, " "))
		cmd = exec.Command(dockerExecCommandNotRunningExecParts[0], dockerExecCommandNotRunningExecParts[1:]...)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}

func main() {
	if len(os.Args) > 1 {
		if len(os.Args) == 2 && os.Args[1] == "--version" {
			fmt.Println(version)
			os.Exit(0)
		}
		fmt.Println(help)
		fmt.Println()
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	var (
		dockerComposeYaml string
		dockerService    string
	)

	allComposeFiles, paths, err := getAllComposeFiles()

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	if len(allComposeFiles) == 0 {
		fmt.Printf("No docker compose YAML found in %s\n", paths)
		os.Exit(1)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Options(huh.NewOptions(allComposeFiles...)...).
				Value(&dockerComposeYaml).
				Title("Docker Compose YAML"),
			huh.NewSelect[string]().
				Value(&dockerService).
				TitleFunc(func() string {
					return "Services in " + dockerComposeYaml
				}, &dockerComposeYaml).
				OptionsFunc(func() []huh.Option[string] {
					serviceKeys, err := getDockerServiceArray(dockerComposeYaml)
					if err != nil {
						serviceKeys = []string{"Error: " + err.Error()}
					}
					return huh.NewOptions(serviceKeys...)
				}, &dockerComposeYaml /* only this function when `dockerComposeYaml` changes */),
		),
	)

	err = form.Run()
	if err != nil {
		if err.Error() == "user aborted" {
			fmt.Printf("Script terminated by User\n")
			os.Exit(130)
		}
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}

	err = runDockerExec(dockerComposeYaml, dockerService)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
