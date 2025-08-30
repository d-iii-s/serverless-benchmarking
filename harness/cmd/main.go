package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func clearResources(cli *client.Client) {
	for _, workloadCreateResponse := range workloadCreateResponses {
		if workloadCreateResponse.ID != "" {
			err := cli.ContainerRemove(context.Background(), workloadCreateResponse.ID, container.RemoveOptions{
				Force: true, // force removal of the resource if it is running
			})
			if err != nil {
				log.Fatalf("Error removing resource: %v", err)
			}
			log.Printf("Container with ID: %s removed successfully\n", workloadCreateResponse.ID)
		}
	}
	// Stop and delete container
	if benchmarkCreateResponse.ID != "" {
		err := cli.ContainerStop(context.Background(), benchmarkCreateResponse.ID, container.StopOptions{})
		err = cli.ContainerRemove(context.Background(), benchmarkCreateResponse.ID, container.RemoveOptions{
			//Force: true, // force removal of the resource if it is running
		})
		if err != nil {
			log.Fatalf("Error removing resource: %v", err)
		}
		log.Printf("Container with ID: %s removed successfully\n", benchmarkCreateResponse.ID)
	}
	if networkCreateResponse.ID != "" {
		err := cli.NetworkRemove(context.Background(), networkCreateResponse.ID)
		if err != nil {
			log.Fatalf("Error removing network: %v", err)
		}
		log.Printf("Network with ID: %s removed successfully\n", networkCreateResponse.ID)
	}
}

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

func setOptionalNames(cfg *Config, configPath *string) {
	// Use hash (first 6 letters) of the config file as a base for naming
	var hash = fileSHA256(*configPath)[:6]
	if cfg.NetworkName == nil {
		defaultNetworkName := fmt.Sprintf("%s-network", hash)
		cfg.NetworkName = &defaultNetworkName
	}
	if cfg.BenchmarkContainerName == nil {
		defaultBenchmarkContainerName := fmt.Sprintf("%s-container", hash)
		cfg.BenchmarkContainerName = &defaultBenchmarkContainerName
	}
	if cfg.JavaOptions == nil {
		cfg.JavaOptions = new(string)
	}
}

func parseApi(ctx context.Context, apiCfg *ApiConfigList, apiPath string) (*openapi3.T, error) {
	loader := &openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
	return loader.LoadFromFile(apiPath)
}

func parseQueryInfo(config *ApiConfig, doc *openapi3.T) (*QueryInfo, error) {
	queryInfo := &QueryInfo{}

	//parse port
	for _, server := range doc.Servers {
		// search for localhost server with specified port as variable
		if server.URL != "" && server.URL >= "http://localhost:{port}" {
			// check if port specified
			if server.Variables == nil || server.Variables["port"] == nil {
				return nil, fmt.Errorf("server URL '%s' does not have 'port' variable defined", server.URL)
			}
			queryInfo.Port = server.Variables["port"].Default
		}
	}

	if queryInfo.Port == "" {
		return nil, errors.New("no port found in API documentation")
	}

	for rawPath, pathItem := range doc.Paths.Map() {
		for operations, operationsItem := range pathItem.Operations() {
			if operationsItem.RequestBody == nil || operationsItem.RequestBody.Value == nil || operationsItem.RequestBody.Value.Content == nil {
				continue
			}
			for applicationType, content := range operationsItem.RequestBody.Value.Content {
				example, err := content.Examples.JSONLookup(config.ExampleName)
				if example != nil && err == nil {
					// resolve header
					if queryInfo.Header == nil {
						queryInfo.Header = make(map[string][]string)
					}
					// resolve path or query variables
					queryInfo.Endpoint = rawPath
					for _, parameterItem := range operationsItem.Parameters {
						if parameterItem.Value.In == "query" || parameterItem.Value.In == "path" {
							queryInfo.Endpoint = strings.Replace(queryInfo.Endpoint, "{"+parameterItem.Value.Name+"}", fmt.Sprintf("%v", parameterItem.Value.Example), -1)
						} else if parameterItem.Value.In == "header" {
							queryInfo.Header[parameterItem.Value.Name] = []string{fmt.Sprintf("%v", parameterItem.Value.Example)}
						}
					}
					// resolve method
					queryInfo.Method = operations
					// resolve body
					if example.(*openapi3.Example).Value != nil {
						queryInfo.Body = example.(*openapi3.Example).Value.(string)
					} else {
						queryInfo.FilePath = example.(*openapi3.Example).ExternalValue
					}
					queryInfo.Header["Content-Type"] = []string{applicationType}
				}
			}
		}
	}

	if queryInfo.Body == "" && queryInfo.FilePath == "" {
		return nil, errors.New("no body or file path found in API documentation")
	}

	return queryInfo, nil
}

var networkCreateResponse network.CreateResponse
var benchmarkCreateResponse container.CreateResponse
var workloadCreateResponses = make([]container.CreateResponse, 0)

func main() {
	var configPath = flag.String("config-path", "config.json", "Path to config file")
	flag.Parse()

	cfg := &Config{}
	if err := parseJSONFile(*configPath, cfg); err != nil {
		log.Fatal(err)
	}

	setOptionalNames(cfg, configPath)

	workdirPath := createWorkdir(cfg.ResultPath)
	createJFRWorkdir(workdirPath)

	benchmarkApiPath := filepath.Join(cfg.BenchmarksRootPath, cfg.BenchmarkName, "api")

	// Parse API configs
	apiConfigsPath := filepath.Join(benchmarkApiPath, "config.json")
	apiConfigList := ApiConfigList{}
	if err := parseJSONFile(apiConfigsPath, &apiConfigList); err != nil {
		log.Fatal(err)
	}

	// Parse API documentation
	apiDocPath := filepath.Join(benchmarkApiPath, "api.yaml")
	apiDoc, err := parseApi(context.Background(), &apiConfigList, apiDocPath)
	if err != nil {
		log.Fatalf("Error parsing API documentation: %v", err)
	}

	apiConfig := apiConfigList.FindByName(cfg.BenchmarkConfigName)
	for _, wrkScript := range apiConfig.WrkScripts {
		if err := CopyFilePreserveName(filepath.Join(benchmarkApiPath, wrkScript), workdirPath); err != nil {
			log.Panicf("Error copying script to mount path: %v", err)
		}
	}

	// copy parser lua framework to workdir
	CopyFilePreserveName(filepath.Join(cfg.BenchmarksRootPath, "lib/parser.lua"), workdirPath)

	// copy api.yaml to workdir
	CopyFilePreserveName(apiDocPath, workdirPath)

	// copy resources to workdir
	CopyDir(filepath.Join(benchmarkApiPath, "resources"), workdirPath, &[]string{})

	queryInfo, err := parseQueryInfo(apiConfig, apiDoc)
	if err != nil {
		log.Fatalf("Error parsing query info: %v", err)
	}
	// client.WithHost("unix:///home/bakhtia/.docker/desktop/docker.sock")
	// Create docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Panicf("Error creating docker client: %v", err)
	}
	log.Printf("Connected to docker host at %s with client version - %s", cli.DaemonHost(), cli.ClientVersion())

	// Clear created resources after execution.
	defer clearResources(cli)

	// Create private network for containers communication.
	networkCreateResponse, err = cli.NetworkCreate(context.Background(), *cfg.NetworkName, network.CreateOptions{
		Driver: "bridge",
	})
	if err != nil {
		log.Panicf("Error creating network: %v", err)
	}
	log.Printf("Network created with ID: %s\n", networkCreateResponse.ID)

	// Create benchmark container.
	benchmarkCreateResponse, err = createBenchmarkContainer(context.Background(), cli, cfg.BenchmarkImage, networkCreateResponse.ID,
		queryInfo.Port, cfg.HostPort, workdirPath, *cfg.BenchmarkContainerName, workdirPath, *cfg.JavaOptions)
	if err != nil {
		log.Panicf("Error creating benchmark container: %v", err)
	}
	log.Printf("Benchmark container created with ID: %s\n", benchmarkCreateResponse.ID)

	if !isPortAvailable(cfg.HostPort) {
		log.Panicf("Port %s on host is not available", cfg.HostPort)
	}

	// start benchmark container
	var benchmarkContainerStartTime = time.Now()
	if err := cli.ContainerStart(context.Background(), benchmarkCreateResponse.ID, container.StartOptions{}); err != nil {
		log.Panicf("Error starting benchmark container: %v", err)
	}
	log.Printf("Benchmark container with ID: %s started successfully\n", benchmarkCreateResponse.ID)

	var requestTime, responseTime = measureFirstTimeRequest(*queryInfo, cfg.HostPort, workdirPath)

	err = WriteCSV(filepath.Join(workdirPath, "first-response.csv"),
		[]string{"benchmark_container_start_timestamp", "request_timestamp", "response_timestamp"},
		[][]string{
			{benchmarkContainerStartTime.Format(time.StampMilli), requestTime.Format(time.StampMilli), responseTime.Format(time.StampMilli)},
		},
	)
	if err != nil {
		log.Panicf("Error writing csv: %v", err)
	}

	go collectContainerStats(context.Background(), cli, benchmarkCreateResponse.ID, workdirPath)
	go measureRSS(context.Background(), cli, benchmarkCreateResponse.ID, workdirPath)

	for idx, wrkScript := range apiConfig.WrkScripts {
		// create workload container
		url := fmt.Sprintf("http://%s:%s%s", *cfg.BenchmarkContainerName, queryInfo.Port, queryInfo.Endpoint)
		workloadCreateResponse, err := createWorkloadContainer(context.Background(), cli, cfg.WorkloadImage, networkCreateResponse.ID, workdirPath,
			cfg.Wrk2Params, url, apiConfig.ExampleName, wrkScript, strconv.Itoa(idx))
		if err != nil {
			log.Panicf("Error creating workload container: %v", err)
		}
		log.Printf("Workload container created: %s, ID: %s\n", cfg.WorkloadImage, workloadCreateResponse.ID)
		workloadCreateResponses = append(workloadCreateResponses, workloadCreateResponse)

		// start workload container
		log.Println("Starting workload container...")
		if err := cli.ContainerStart(context.Background(), workloadCreateResponse.ID, container.StartOptions{}); err != nil {
			log.Panicf("Error starting workload container: %v", err)
		}

		statusCh, errCh := cli.ContainerWait(context.Background(), workloadCreateResponse.ID, container.WaitConditionNotRunning)
		if err != nil {
			log.Panicf("Error running workload container: %v", err)
		}

		select {
		case status := <-statusCh:
			log.Printf("Workload container exited with status code: %d\n", status.StatusCode)
		case err := <-errCh:
			log.Panic("error waiting for container: %w", err)
		}
	}
}
