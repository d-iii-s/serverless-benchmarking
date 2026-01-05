package docker

import (
	"archive/tar"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/d-iii-s/slsbench/internal/model"
	"github.com/d-iii-s/slsbench/internal/utils"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const WorkingDir string = "/wrktemp"
const OutputDirContainer string = "/results"

func CreateWorkloadContainerV2(ctx context.Context, cli *client.Client, imageName, serviceName, wrk2params, resultDir, scenarioPath string, port int) (container.CreateResponse, error) {
	env := []string{}
	if serviceName != "" {
		env = append(env, fmt.Sprintf("SERVICE_NAME=%s", serviceName))
	}
	if wrk2params != "" {
		env = append(env, fmt.Sprintf("WRK2PARAMS=%s", wrk2params))
	}
	if port > 0 {
		env = append(env, fmt.Sprintf("PORT=%d", port))
	}
	env = append(env, fmt.Sprintf("OUTPUT_DIR=%s", WorkingDir+OutputDirContainer))

	containerConfig := &container.Config{
		Image: imageName,
		//Cmd:   []string{"sh", "-c", "sleep infinity"},
		Cmd: []string{"sh", "-c", fmt.Sprintf("cd %s; ./run_benchmark.sh", WorkingDir)},
		Env: env,
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: scenarioPath,
				Target: WorkingDir + "/scenario.json",
			},
			{
				Type:   mount.TypeBind,
				Source: resultDir,
				Target: WorkingDir + OutputDirContainer,
			},
		},
	}
	networkConfig := &network.NetworkingConfig{}
	return cli.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, "")
}

func CollectContainerStats(ctx context.Context, cli *client.Client, containerId string, workingDir string) {
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
	//writer.Write([]string{"timestamp", "pid_count", "pid_count_Limit", "cpu_usage", "memory_usage", "rx_bytes", "tx_bytes"})
	writer.Write([]string{"timestamp", "pid_count", "pid_count_Limit", "cpu_usage", "memory_usage"})
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
			//var rxBytes = fmt.Sprintf("%d", firstNetwork(stats).RxBytes)
			//var txBytes = fmt.Sprintf("%d", firstNetwork(stats).TxBytes)
			//err := writer.Write([]string{time.Now().String(), PIDs, PIDsLimit, cpuUsage, memoryUsage, rxBytes, txBytes})
			err := writer.Write([]string{time.Now().String(), PIDs, PIDsLimit, cpuUsage, memoryUsage})
			if err != nil {
				log.Println("Error creating csv:", err)
			}
			writer.Flush()
		}
	}
}

// CopyFromContainer copies a file or directory from a container to the host filesystem.
// containerPath is the path inside the container to copy from.
// destPath is the directory on the host where the files will be extracted.
func CopyFromContainer(ctx context.Context, cli *client.Client, containerID, containerPath, destPath string) error {
	// Get the content from the container as a tar stream
	reader, stat, err := cli.CopyFromContainer(ctx, containerID, containerPath)
	if err != nil {
		return fmt.Errorf("failed to copy from container: %w", err)
	}
	defer reader.Close()

	// Create a subdirectory in destPath based on the container path
	// to avoid overwriting existing files
	baseName := filepath.Base(containerPath)
	if stat.Name != "" {
		baseName = stat.Name
	}

	// Create "collected" subdirectory to organize collected files
	collectedDir := filepath.Join(destPath, "collected")
	if err := os.MkdirAll(collectedDir, 0755); err != nil {
		return fmt.Errorf("failed to create collected directory: %w", err)
	}

	// Extract the tar archive
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Determine the target path
		// Remove leading slash and use the base name as prefix
		cleanName := strings.TrimPrefix(header.Name, "/")
		targetPath := filepath.Join(collectedDir, cleanName)

		// Ensure the target path is within the destination directory (security check)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(collectedDir)) {
			log.Printf("Warning: skipping potentially unsafe path: %s", header.Name)
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}

		case tar.TypeReg:
			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories for %s: %w", targetPath, err)
			}

			// Create and write file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				log.Printf("Warning: failed to set permissions on %s: %v", targetPath, err)
			}

		case tar.TypeSymlink:
			// Create symlink
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directories for symlink %s: %w", targetPath, err)
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				log.Printf("Warning: failed to create symlink %s: %v", targetPath, err)
			}

		default:
			log.Printf("Warning: unsupported file type for %s (type: %c)", header.Name, header.Typeflag)
		}
	}

	log.Printf("Extracted %s to %s", baseName, collectedDir)
	return nil
}

// https://quarkus.io/guides/performance-measure#measuring-memory-correctly-on-docker
func MeasureRSS(ctx context.Context, cli *client.Client, containerID, workdir string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var allSnapshots []model.Snapshot

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

			snapshot := model.Snapshot{
				Timestamp: time.Now(),
				Processes: []model.ProcessInfo{},
			}

			for _, proc := range top.Processes {
				if len(proc) >= 3 {
					snapshot.Processes = append(snapshot.Processes, model.ProcessInfo{
						PID:  proc[0],
						RSS:  proc[1],
						Args: proc[2],
					})
				}
			}

			allSnapshots = append(allSnapshots, snapshot)

			// Save after every new snapshot
			utils.SaveData(allSnapshots, filepath.Join(workdir, "rss_info.json"))
		}
	}
}
