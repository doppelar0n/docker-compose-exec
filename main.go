package main

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var DEFAULT_CONTAINER_BASE_PATH = "/var/container:/srv/container"
var DEFAULT_CONTAINER_EXEC_COMMAND = "docker compose -f %COMPOSE exec --user root %SERVICE /bin/sh"

//go:embed HELP.md
var help string

var version = "development"

func isDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func getSubdirectories(basePath string) ([]string, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	var subdirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			subdirs = append(subdirs, filepath.Join(basePath, entry.Name()))
		}
	}
	return subdirs, nil
}

func getComposeFileInDir(dir string) (string, error) {
	composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, file := range composeFiles {
		fullPath := filepath.Join(dir, file)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
	}
	return "", fmt.Errorf("no Docker Compose in %s found", dir)
}

func getComposeFiles(basePath string) ([]string, error) {
	isDir, err := isDirectory(basePath)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, fmt.Errorf("%s is not a Directory.", basePath)
	}

	subdirs, err := getSubdirectories(basePath)
	if err != nil {
		return nil, err
	}

	var composeFiles []string
	for _, dir := range subdirs {
		composeFile, err := getComposeFileInDir(dir)
		if err == nil {
			composeFiles = append(composeFiles, composeFile)
		}
	}

	return composeFiles, nil
}

func getAllComposeSearchPaths() []string {
	var paths []string

	containerBasePathString := os.Getenv("CONTAINER_BASE_PATH")
	if containerBasePathString == "" {
		containerBasePathString = DEFAULT_CONTAINER_BASE_PATH
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

	for _, path := range paths {
		currentComposeFilePaths, _ := getComposeFiles(path)
		composeFilePaths = append(composeFilePaths, currentComposeFilePaths...)
	}

	return composeFilePaths, strings.Join(paths, " "), nil
}

func getDockerServiceArray(dockerComposeYml string) ([]string, error) {
	var serviceKeys []string

	data, err := os.ReadFile(dockerComposeYml)
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
		return nil, errors.New("rawData[\"services\"].(map[string]interface{}) is no ok!")
	}

	for key := range services {
		serviceKeys = append(serviceKeys, key)
	}

	return serviceKeys, nil
}

func runDockerExec(dockerComposeYml string, dockerService string) error {
	dockerExecCommand := os.Getenv("CONTAINER_EXEC_COMMAND")
	if dockerExecCommand == "" {
		dockerExecCommand = DEFAULT_CONTAINER_EXEC_COMMAND
	}

	dockerExecCommandParts := strings.Split(dockerExecCommand, " ")
	for i, part := range dockerExecCommandParts {
		if part == "%COMPOSE" {
			dockerExecCommandParts[i] = dockerComposeYml
		} else if part == "%SERVICE" {
			dockerExecCommandParts[i] = dockerService
		}
	}

	fmt.Printf("dockerExecCommand: %s\n", dockerExecCommand)
	fmt.Printf("runDockerCommand:  %s\n", strings.Join(dockerExecCommandParts, " "))

	cmd := exec.Command(dockerExecCommandParts[0], dockerExecCommandParts[1:]...)
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
		fmt.Println("Version: " + version)
		os.Exit(0)
	}

	var (
		dockerComposeYml string
		dockerService    string
	)

	allComposeFiles, paths, err := getAllComposeFiles()

	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	if len(allComposeFiles) == 0 {
		fmt.Println("Error: no docker compose.yml found in ", paths)
		os.Exit(1)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Options(huh.NewOptions(allComposeFiles...)...).
				Value(&dockerComposeYml).
				Title("Docker Compose yml").
				Height(5),
			huh.NewSelect[string]().
				Value(&dockerService).
				Height(8).
				TitleFunc(func() string {
					return "Services in " + dockerComposeYml
				}, &dockerComposeYml).
				OptionsFunc(func() []huh.Option[string] {
					serviceKeys, err := getDockerServiceArray(dockerComposeYml)
					if err != nil {
						serviceKeys = []string{"Error: " + err.Error()}
					}
					return huh.NewOptions(serviceKeys...)
				}, &dockerComposeYml /* only this function when `dockerComposeYml` changes */),
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

	err = runDockerExec(dockerComposeYml, dockerService)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
