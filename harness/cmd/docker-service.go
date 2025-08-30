package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const CONTAINER_MOUNT_PATH string = "/data"

func createBenchmarkContainer(ctx context.Context, cli *client.Client, imageName, networkId, containerPort, hostPort,
	mountPath, containerName, workdirPath, javaOpts string) (container.CreateResponse, error) {
	containerConfig := &container.Config{
		Image: imageName,
		ExposedPorts: nat.PortSet{
			nat.Port(containerPort): struct{}{},
		},
		Env: []string{
			fmt.Sprintf("HARNESS_JAVA_OPTS=%s", javaOpts),
		},
	}
	var absMountPath, _ = filepath.Abs(mountPath)
	var absWorkdirPath, _ = filepath.Abs(workdirPath)
	var mounts = []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: absMountPath,
			Target: CONTAINER_MOUNT_PATH,
		},
		// by default jfr logs are mounted
		{
			Type:   mount.TypeBind,
			Source: filepath.Join(absWorkdirPath, "jfr"),
			Target: "/app/logs",
		},
	}

	// enable container writing to jfr directory
	if err := os.Chmod(filepath.Join(absWorkdirPath, "jfr"), 0777); err != nil {
		log.Printf("Failed to set permissions for jfr directory: %v", err)
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(containerPort): []nat.PortBinding{{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			}},
		},
		Mounts: mounts,
	}
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkId: {
				NetworkID: networkId,
			},
		},
	}
	return cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
}

func createWorkloadContainer(ctx context.Context, cli *client.Client, imageName, networkId, mountPath, wrk2params, url, sampleName,
	scriptName, outputPrefix string) (container.CreateResponse, error) {
	var outputFileName = fmt.Sprintf("%s-%s.out", outputPrefix, scriptName)
	var command = fmt.Sprintf("cd /data && wrk2 %s -s %s %s > %s", wrk2params, scriptName, url, outputFileName)
	containerConfig := &container.Config{
		Image: imageName,
		Cmd:   []string{"sh", "-c", command},
		Env: []string{
			fmt.Sprintf("OUTPUT_DIR=%s", CONTAINER_MOUNT_PATH),
			fmt.Sprintf("HOST_URL=%s", url),
			fmt.Sprintf("SAMPLE_NAME=%s", sampleName),
		},
	}
	var absMountPath, _ = filepath.Abs(mountPath)
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{{
			Type:   mount.TypeBind,
			Source: absMountPath,
			Target: CONTAINER_MOUNT_PATH,
		}},
	}
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkId: {
				NetworkID: networkId,
			},
		},
	}
	return cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, "")
}

func measureFirstTimeRequest(queryInfo QueryInfo, benchmarkExposedPort, workdirPath string) (time.Time, time.Time) {
	log.Println("Measuring first time request...")
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var reqTime = time.Time{}
	var respTime = time.Time{}
	var isSuccessful = true
	wg.Add(1)

	go func() {
		defer wg.Done()
		httpClient := &http.Client{}
		for {
			currReqTime := time.Now()
			select {
			case <-ctx.Done():
				return // exit async function - timeout
			default:
				// injecting domain and benchmark port into url string
				parsedURL, err := url.Parse(fmt.Sprintf("http://localhost:%s%s", benchmarkExposedPort, queryInfo.Endpoint))
				if err != nil {
					panic(err)
				}

				var data io.ReadCloser
				if queryInfo.FilePath != "" {
					// Open the file in binary read mode
					fileBytes, err := os.ReadFile(filepath.Join(workdirPath, queryInfo.FilePath))
					if err != nil {
						log.Fatal("", err)
					}
					data = io.NopCloser(bytes.NewReader(fileBytes))
				} else {
					data = io.NopCloser(strings.NewReader(queryInfo.Body))
				}

				req := &http.Request{
					Method: queryInfo.Method,
					URL:    parsedURL,
					Header: queryInfo.Header,
					Body:   data,
				}
				req.WithContext(ctx)

				res, err := httpClient.Do(req)

				if res != nil && res.StatusCode == 200 {
					reqTime = currReqTime
					respTime = time.Now()
					cancel()
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case <-time.After(30 * time.Second):
		isSuccessful = false
		cancel()
	}

	wg.Wait()

	if !isSuccessful {
		log.Panicf("First time request measurment does not complete succesfully ...")
	} else {
		log.Println("First time request successfully measured")
	}
	return reqTime, respTime
}

func firstNetwork(stats container.StatsResponse) container.NetworkStats {
	for _, v := range stats.Networks {
		return v // first network interface
	}
	return container.NetworkStats{}
}

func collectContainerStats(ctx context.Context, cli *client.Client, containerId string, workingDir string) {
	log.Println("Collecting container stats...")
	statsResp, err := cli.ContainerStats(ctx, containerId, true)
	if err != nil {
		log.Panic(err)
	}
	defer statsResp.Body.Close()
	decoder := json.NewDecoder(statsResp.Body)
	// create or open the CSV file
	file, err := os.Create(filepath.Join(workingDir, "container-stats.csv"))
	if err != nil {
		log.Panic(err)
	}
	writer := csv.NewWriter(file)
	defer file.Close()
	// write header
	writer.Write([]string{"timestamp", "pid_count", "pid_count_Limit", "cpu_usage", "memory_usage", "rx_bytes", "tx_bytes"})
	writer.Flush()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			var stats container.StatsResponse
			// open file and do not close and write value to it on each stats arrived
			if err := decoder.Decode(&stats); err != nil {
				if err == io.EOF {
					log.Println("Stream ended.")
					return
				}
				log.Println("Decode error:", err)
				continue
			}

			var PIDs = fmt.Sprintf("%d", stats.PidsStats.Current)
			var PIDsLimit = fmt.Sprintf("%d", stats.PidsStats.Limit)
			var cpuUsage = fmt.Sprintf("%d", stats.CPUStats.CPUUsage.TotalUsage)
			var memoryUsage = fmt.Sprintf("%d", stats.MemoryStats.Usage)
			var rxBytes = fmt.Sprintf("%d", firstNetwork(stats).RxBytes)
			var txBytes = fmt.Sprintf("%d", firstNetwork(stats).TxBytes)
			err := writer.Write([]string{time.Now().String(), PIDs, PIDsLimit, cpuUsage, memoryUsage, rxBytes, txBytes})
			if err != nil {
				log.Println("Error creating csv:", err)
			}
			writer.Flush()
		}
	}
}

// https://quarkus.io/guides/performance-measure#measuring-memory-correctly-on-docker
func measureRSS(ctx context.Context, cli *client.Client, containerID, workdir string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var allSnapshots []Snapshot

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping monitoring.")
			return

		case <-ticker.C:
			top, err := cli.ContainerTop(ctx, containerID, []string{"-o", "pid,rss,args"})
			if err != nil {
				log.Println("Error fetching top:", err)
				continue
			}

			snapshot := Snapshot{
				Timestamp: time.Now(),
				Processes: []ProcessInfo{},
			}

			for _, proc := range top.Processes {
				if len(proc) >= 3 {
					snapshot.Processes = append(snapshot.Processes, ProcessInfo{
						PID:  proc[0],
						RSS:  proc[1],
						Args: proc[2],
					})
				}
			}

			allSnapshots = append(allSnapshots, snapshot)

			// Save after every new snapshot
			SaveData(allSnapshots, filepath.Join(workdir, "rss_info.json"))
		}
	}
}
