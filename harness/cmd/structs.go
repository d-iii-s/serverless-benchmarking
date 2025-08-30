package main

import "time"

type Config struct {
	WorkloadImage  string `json:"workloadImage"`
	BenchmarkName  string `json:"benchmarkName"`
	BenchmarkImage string `json:"benchmarkImage"`
	HostPort       string `json:"hostPort"`
	// Path where results of harness execution will be saved.
	ResultPath string `json:"resultPath"`
	// Path to the benchmark folder with workloads.
	BenchmarksRootPath  string `json:"benchmarksRootPath"`
	BenchmarkConfigName string `json:"benchmarkConfigName"`
	Wrk2Params          string `json:"wrk2params"`
	/*
		OPTIONAL
	*/
	BenchmarkContainerName *string `json:"benchmarkContainerName"`
	JavaOptions            *string `json:"javaOptions"`
	// If field is set, then new network for workload and benchmark container communication will be created with given name.
	NetworkName *string `json:"networkName"`
}

type ApiConfig struct {
	Name        string   `json:"name"`
	ExampleName string   `json:"exampleName"`
	WrkScripts  []string `json:"wrkScripts"`
}

// ApiConfigList is a slice of ApiConfig with helper methods.
type ApiConfigList []ApiConfig

type QueryInfo struct {
	Method   string
	Endpoint string
	Body     string
	Port     string
	/**
	Body can be also a file. Body field would be ignored when this field is specified.
	*/
	FilePath string
	Header   map[string][]string
}

// ProcessInfo holds information about one process
type ProcessInfo struct {
	PID  string `json:"pid"`
	RSS  string `json:"rss"`
	Args string `json:"args"`
}

// Snapshot holds information collected at a timestamp
type Snapshot struct {
	Timestamp time.Time     `json:"timestamp"`
	Processes []ProcessInfo `json:"processes"`
}
