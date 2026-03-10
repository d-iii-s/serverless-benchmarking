package model

import (
	"github.com/manifoldco/promptui"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

type PromptSelector interface {
	Run() (int, string, error)
}

type promptuiSelectWrapper struct {
	promptSelect *promptui.Select
}

func (w *promptuiSelectWrapper) Run() (int, string, error) {
	return w.promptSelect.Run()
}

func NewPromptSelector(label string, items []string) PromptSelector {
	return &promptuiSelectWrapper{
		promptSelect: &promptui.Select{
			Label: label,
			Items: items,
		},
	}
}

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

// DataModel represents the complete tree structure of endpoints, operations, and data structures
type DataModel struct {
	Endpoints map[string]*Endpoint `json:"endpoints"` // key: path name
}

// Endpoint represents a single API endpoint (path)
type Endpoint struct {
	Path       string                `json:"path"`       // e.g., "/owners/{ownerId}"
	Operations map[string]*Operation `json:"operations"` // key: HTTP method (get, post, etc.)
}

// Operation represents a single HTTP operation
type Operation struct {
	Method      string                    `json:"method"`                // HTTP method (get, post, put, delete, etc.)
	Parameters  []*Field                  `json:"parameters"`            // Query and path parameters
	RequestBody *DataStructure            `json:"requestBody,omitempty"` // Request body data structure
	Responses   map[string]*DataStructure `json:"responses"`             // key: status code (e.g., "200", "404")
}

// DataStructure represents a data structure (schema) with its fields
type DataStructure struct {
	Name        string   `json:"name"`          // Schema name or path identifier
	Ref         string   `json:"ref,omitempty"` // $ref if this is a component reference
	ContentType string   `json:"contentType"`   // Content type (e.g., "application/json")
	Fields      []*Field `json:"fields"`        // Fields in this data structure
}

// Field represents a single field in a data structure
type Field struct {
	Name       string            `json:"name"`                 // Field name
	Type       string            `json:"type"`                 // Field type (string, integer, object, array, etc.)
	Format     string            `json:"format,omitempty"`     // Format (e.g., "date", "int32")
	Required   bool              `json:"required"`             // Whether field is required
	Hint       string            `json:"hint,omitempty"`       // Hint from x-user-hint extension
	Unique     bool              `json:"unique,omitempty"`     // Whether field is unique (from x-slsbench-unique extension)
	Min        *float64          `json:"min,omitempty"`         // Minimum value (for numbers/integers)
	Max        *float64          `json:"max,omitempty"`         // Maximum value (for numbers/integers)
	MinLength  uint64            `json:"minLength,omitempty"`  // Minimum length (for strings)
	MaxLength  *uint64           `json:"maxLength,omitempty"`  // Maximum length (for strings)
	Pattern    string            `json:"pattern,omitempty"`     // Pattern (regex) for string validation
	Path       string            `json:"path"`                 // Full path to this field (e.g., "/owners/{ownerId}/get/requestBody/application/json.firstName")
	Ref        string            `json:"ref,omitempty"`        // $ref if this field references another schema
	Items      *Field            `json:"items,omitempty"`      // For array types, the item schema
	Properties map[string]*Field `json:"properties,omitempty"` // For object types, nested properties
	Schema     *openapi3.Schema  `json:"-"`                    // Original schema reference (excluded from JSON to avoid circular references)
	In         string            `json:"in,omitempty"`         // Parameter location: "query", "path", "header", "cookie" (only for parameters)
}
