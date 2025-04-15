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

var defaultContainerBasePath = "/var/container:/srv/container"
var defaultContainerExecCommand = "docker compose -f %COMPOSE exec --user root %SERVICE /bin/sh"

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

func getComposeFilesInDir(basePath string) ([]string, error) {
	isDir, err := isDirectory(basePath)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, fmt.Errorf("%s is not a Directory", basePath)
	}

	composeFileNames := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	var composeFiles []string
	
	err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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
		return nil, fmt.Errorf("no Docker Compose files found in %s", basePath)
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

	for _, path := range paths {
		currentComposeFilePaths, _ := getComposeFilesInDir(path)
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

func runDockerExec(dockerComposeYaml string, dockerService string) error {
	dockerExecCommand := os.Getenv("CONTAINER_EXEC_COMMAND")
	if dockerExecCommand == "" {
		dockerExecCommand = defaultContainerExecCommand
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
		fmt.Printf("Error: no docker Docker Compose YAML found in %s\n", paths)
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
