package harness

import (
	"context"
	"fmt"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/d-iii-s/slsbench/internal/service/docker"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"log"
	"os"
	"path/filepath"
	"time"
)

func createWorkdir(mountPath string) string {
	// Workdir naming pattern: stdLongYear-stdZeroMonth-stdZeroDay-stdHour:stdZeroMinute:stdZeroSecond
	workdirPath := filepath.Join(mountPath, "result-"+time.Now().Format("2006-01-02-15:04:05"))
	err := os.MkdirAll(workdirPath, 0755)
	if err != nil {
		log.Panicf("Error creating workdir: %v", err)
	}
	return workdirPath
}

func createJFRWorkdir(workdirPath string) {
	err := os.MkdirAll(filepath.Join(workdirPath, "jfr"), 0755)
	if err != nil {
		log.Panicf("Error creating jfr subreddit: %v", err)
	}
}

type EventProcessor struct{}

func newEventProcessor() EventProcessor {
	return EventProcessor{}
}

func (e EventProcessor) Start(ctx context.Context, operation string) {
	log.Printf("Starting operation: %s\n", operation)
}

func (e EventProcessor) On(events ...api.Resource) {
	for _, event := range events {
		log.Printf("Resource ID: %s, Status: %s, Details: %s, Progress: %d/%d (%d%%)\n",
			event.ID, event.Status, event.Details, event.Current, event.Total, event.Percent)
	}
}

func (e EventProcessor) Done(operation string, success bool) {
	if success {
		log.Printf("Operation %s completed successfully\n", operation)
	} else {
		log.Printf("Operation %s failed\n", operation)
	}
}

func Run(ctx context.Context, scenarioPath, wrk2params, resultPath, dockerComposePath, serviceName string, port int, collectPaths []string) {
	// work dir on local machine
	workdirPath := createWorkdir(resultPath)

	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		log.Panicf("Error creating docker CLI: %v", err)
	}

	if err := dockerCLI.Initialize(&flags.ClientOptions{}); err != nil {
		log.Fatalf("Failed to initialize docker CLI: %v", err)
	}

	// new compose service instance
	service, err := compose.NewComposeService(dockerCLI, compose.WithEventProcessor(newEventProcessor()))
	if err != nil {
		log.Panicf("Error creating compose service: %v", err)
	}

	// load the compose project from a compose file
	project, err := service.LoadProject(ctx, api.ProjectLoadOptions{
		ConfigPaths: []string{dockerComposePath},
		ProjectName: fmt.Sprintf("benchmarking-application-%d", time.Now().Unix()),
	})
	if err != nil {
		log.Fatalf("Failed to load project: %v", err)
	}

	// Create Docker client early so we can create workload container
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Panicf("Error creating docker client: %v", err)
	}

	// Create workload container before starting services
	workloadContainerResponse, err := docker.CreateWorkloadContainerV2(ctx, cli, utils.DefaultWorkloadGeneratorImage,
		serviceName, wrk2params, workdirPath, scenarioPath, port)
	if err != nil {
		log.Panicf("Error creating workload container: %v", err)
	}
	log.Printf("Created workload container: %s", workloadContainerResponse.ID)

	// ------------------------------------------------------------
	// CLEAR RESOURCES
	// ------------------------------------------------------------
	defer clearResources(ctx, err, cli, workloadContainerResponse, service, project)

	// Create networks and containers (but don't start them yet)
	// This ensures the network exists so we can connect the workload container
	err = service.Create(ctx, project, api.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create services: %v", err)
	}
	log.Printf("Created project resources: %s", project.Name)

	// Connect workload container to the project's network
	// First, try to get network name from Docker Compose file
	networkName := getNetworkName(project)

	// If no network name found in compose file, fall back to default naming pattern
	if networkName == "" {
		networkName = fmt.Sprintf("%s_default", project.Name)
	}

	networkInfo, err := cli.NetworkInspect(ctx, networkName, network.InspectOptions{})
	if err != nil {
		log.Panicf("Error inspecting network %s: %v", networkName, err)
	}

	err = cli.NetworkConnect(ctx, networkInfo.ID, workloadContainerResponse.ID, nil)
	if err != nil {
		log.Panicf("Error connecting container to network: %v", err)
	}
	log.Printf("Successfully connected workload container to network: %s", networkName)

	go startWorkloadContainer(context.Background(), cli, workloadContainerResponse)

	// Now start the services
	err = service.Start(ctx, project.Name, api.StartOptions{
		Project: project,
	})
	if err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}
	log.Printf("Successfully started project: %s", project.Name)

	// Find container ID of the service
	serviceContainerID, err := findContainerIDByServiceName(ctx, cli, project.Name, serviceName)
	if err != nil {
		log.Panicf("Error finding container for service %s: %v", serviceName, err)
	}
	log.Printf("Found container ID for service %s: %s", serviceName, serviceContainerID)

	go docker.CollectContainerStats(context.Background(), cli, serviceContainerID, workdirPath)
	go docker.MeasureRSS(context.Background(), cli, serviceContainerID, workdirPath)

	// Wait for the workload container to finish
	statusCh, errCh := cli.ContainerWait(ctx, workloadContainerResponse.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			log.Panicf("Error waiting for workload container: %v", err)
		}
	case status := <-statusCh:
		if status.Error != nil {
			log.Panicf("Error in workload container: %v", status.Error.Message)
		}
		if status.StatusCode != 0 {
			log.Printf("Workload container exited with status code: %d", status.StatusCode)
		} else {
			log.Println("Workload container finished successfully")
		}
	}
	time.Sleep(2 * time.Second) // wait for a while to collect final stats

	// Copy specified paths from service container to results
	if len(collectPaths) > 0 {
		log.Printf("Collecting %d path(s) from service container...", len(collectPaths))
		for _, containerPath := range collectPaths {
			if containerPath == "" {
				continue
			}
			log.Printf("Copying %s from container to results...", containerPath)
			if err := docker.CopyFromContainer(ctx, cli, serviceContainerID, containerPath, workdirPath); err != nil {
				log.Printf("Warning: Failed to copy %s: %v", containerPath, err)
			} else {
				log.Printf("Successfully copied %s to results", containerPath)
			}
		}
	}
}

func startWorkloadContainer(ctx context.Context, cli *client.Client, workloadContainerResponse container.CreateResponse) {
	// Start the workload container after services are running
	if err := cli.ContainerStart(ctx, workloadContainerResponse.ID, container.StartOptions{}); err != nil {
		log.Panicf("Error starting workload container: %v", err)
	}
	log.Println("Workload container started successfully")
}

func clearResources(ctx context.Context, err error, cli *client.Client, workloadContainerResponse container.CreateResponse, service api.Compose, project *types.Project) {
	err = cli.ContainerRemove(context.Background(), workloadContainerResponse.ID, container.RemoveOptions{
		Force: true, // force removal of the resource if it is running
	})
	if err != nil {
		log.Fatalf("Error removing resource: %v", err)
	}
	err = service.Down(ctx, project.Name, api.DownOptions{
		Project: project,
	})
	if err != nil {
		log.Fatalf("Failed to remove project resources: %v", err)
	}
}

// findContainerIDByServiceName finds the container ID for a given service name in a Docker Compose project.
// It uses Docker labels to filter containers by project and service name.
func findContainerIDByServiceName(ctx context.Context, cli *client.Client, projectName, serviceName string) (string, error) {
	filterArgs := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", "com.docker.compose.project", projectName)),
		filters.Arg("label", fmt.Sprintf("%s=%s", "com.docker.compose.service", serviceName)),
	)

	containers, err := cli.ContainerList(ctx, container.ListOptions{
		Filters: filterArgs,
		All:     false, // Only running containers
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("no container found for service %s in project %s", serviceName, projectName)
	}

	// Return the first container's ID (if there are multiple, take the first one)
	return containers[0].ID, nil
}

// getNetworkName extracts the network name from the Docker Compose project.
// It checks if any network in the compose file has an explicit 'name' field.
// If no explicit name is found, it returns an empty string to use the default naming pattern.
func getNetworkName(project *types.Project) string {
	// If no networks are defined in the compose file, return empty to use default
	if project.Networks == nil || len(project.Networks) == 0 {
		return ""
	}

	// Check each network for an explicit name field
	for networkKey, networkConfig := range project.Networks {
		// If the network has an explicit name set, use it
		// This works for both regular networks and external networks
		if networkConfig.Name != "" {
			return networkConfig.Name
		}
		// If network is external (bool is true), the name might be the network key
		// but we should still check if there's a name field first (handled above)
		// Otherwise, Docker Compose will use {project-name}_{networkKey}
		_ = networkKey // avoid unused variable warning
	}

	// If we have networks but no explicit names, check which network services are using
	// and use the first one found (Docker Compose creates it as {project-name}_{networkKey})
	// For simplicity, return empty to use the default {project-name}_default pattern
	// The caller will handle the default naming
	return ""
}
