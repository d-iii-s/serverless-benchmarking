package model

import (
	"encoding/json"
	"fmt"
	"github.com/manifoldco/promptui"
	"os"
	"strings"
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

type ScenarioGraph struct {
	vertices map[int]*ScenarioGraphVertex
	edges    []*ScenarioGraphEdge
}

type ScenarioGraphVertex struct {
	Path         string         `json:"path"`
	Method       string         `json:"method"`                // HTTP method (get, post, put, delete, etc.)
	Parameters   []*Field       `json:"parameters"`            // Query and path parameters
	RequestBody  *DataStructure `json:"requestBody,omitempty"` // Request body data structure
	ResponseCode string         `json:"responseCode"`          // e.g., "200", "404"
}

type ScenarioGraphEdge struct {
	From     int               `json:"from"`
	To       int               `json:"to"`
	Mappings map[string]string `json:"mappings"` // key: source field path, value: target field path
}

// NewScenarioGraph creates a new empty ScenarioGraph
func NewScenarioGraph() *ScenarioGraph {
	return &ScenarioGraph{
		vertices: make(map[int]*ScenarioGraphVertex),
		edges:    []*ScenarioGraphEdge{},
	}
}

// AddVertex adds a new vertex to the graph and returns its ID
func (g *ScenarioGraph) AddVertex(vertex *ScenarioGraphVertex) int {
	id := len(g.vertices)
	g.vertices[id] = vertex
	return id
}

// AddEdge adds a new edge between two vertices
func (g *ScenarioGraph) AddEdge(from, to int) *ScenarioGraphEdge {
	edge := &ScenarioGraphEdge{
		From:     from,
		To:       to,
		Mappings: make(map[string]string),
	}
	g.edges = append(g.edges, edge)
	return edge
}

// GetVertex returns a vertex by ID
func (g *ScenarioGraph) GetVertex(id int) *ScenarioGraphVertex {
	return g.vertices[id]
}

// GetVertices returns all vertices
func (g *ScenarioGraph) GetVertices() map[int]*ScenarioGraphVertex {
	return g.vertices
}

// GetEdges returns all edges
func (g *ScenarioGraph) GetEdges() []*ScenarioGraphEdge {
	return g.edges
}

// GetEdge finds an edge between two vertices
func (g *ScenarioGraph) GetEdge(from, to int) *ScenarioGraphEdge {
	for _, edge := range g.edges {
		if edge.From == from && edge.To == to {
			return edge
		}
	}
	return nil
}

// AddMapping adds a field mapping to an existing edge
func (e *ScenarioGraphEdge) AddMapping(sourceField, targetField string) {
	e.Mappings[sourceField] = targetField
}

// TopologicalSort performs Kahn's algorithm to find a topological ordering of vertices.
// Returns the ordered list of vertex IDs and an error if a cycle is detected.
func (g *ScenarioGraph) TopologicalSort() ([]int, error) {
	// Calculate in-degree for each vertex (number of incoming edges)
	inDegree := make(map[int]int)
	
	// Initialize in-degree for all vertices
	for id := range g.vertices {
		inDegree[id] = 0
	}
	
	// Build adjacency list (for each vertex, which vertices it points to)
	adjacencyList := make(map[int][]int)
	
	// Calculate in-degrees and build adjacency list
	for _, edge := range g.edges {
		// Edge goes from edge.From to edge.To
		// So edge.To has one more incoming edge
		inDegree[edge.To]++
		
		// Add edge.To to the adjacency list of edge.From
		adjacencyList[edge.From] = append(adjacencyList[edge.From], edge.To)
	}
	
	// Queue for vertices with in-degree 0
	queue := []int{}
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	
	// Result: topological order
	result := []int{}
	
	// Process vertices
	for len(queue) > 0 {
		// Dequeue a vertex
		current := queue[0]
		queue = queue[1:]
		
		// Add to result
		result = append(result, current)
		
		// Process all neighbors
		for _, neighbor := range adjacencyList[current] {
			// Reduce in-degree
			inDegree[neighbor]--
			
			// If in-degree becomes 0, add to queue
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}
	
	// Check for cycle: if we haven't processed all vertices, there's a cycle
	if len(result) != len(g.vertices) {
		return nil, fmt.Errorf("cycle detected in scenario graph: cannot determine execution order")
	}
	
	return result, nil
}

// VertexLabel returns a human-readable label for the vertex
func (v *ScenarioGraphVertex) VertexLabel() string {
	return fmt.Sprintf("%s %s [%s]", v.Method, v.Path, v.ResponseCode)
}

// String returns a string representation of the graph for dumping
func (g *ScenarioGraph) String() string {
	var sb strings.Builder
	sb.WriteString("\n========== SCENARIO GRAPH ==========\n")

	sb.WriteString("\n--- VERTICES ---\n")
	if len(g.vertices) == 0 {
		sb.WriteString("  (no vertices)\n")
	} else {
		for id, v := range g.vertices {
			sb.WriteString(fmt.Sprintf("  [%d] %s\n", id, v.VertexLabel()))
		}
	}

	sb.WriteString("\n--- EDGES ---\n")
	if len(g.edges) == 0 {
		sb.WriteString("  (no edges)\n")
	} else {
		for _, e := range g.edges {
			fromLabel := "(unknown)"
			toLabel := "(unknown)"
			if v := g.vertices[e.From]; v != nil {
				fromLabel = v.VertexLabel()
			}
			if v := g.vertices[e.To]; v != nil {
				toLabel = v.VertexLabel()
			}
			sb.WriteString(fmt.Sprintf("  [%d] %s -> [%d] %s\n", e.From, fromLabel, e.To, toLabel))
			if len(e.Mappings) > 0 {
				sb.WriteString("    Mappings:\n")
				for src, dst := range e.Mappings {
					sb.WriteString(fmt.Sprintf("      %s -> %s\n", src, dst))
				}
			}
		}
	}
	sb.WriteString("=====================================\n")
	return sb.String()
}

// scenarioGraphJSON is an exportable struct for JSON serialization
type scenarioGraphJSON struct {
	Vertices map[int]*ScenarioGraphVertex `json:"vertices"`
	Edges    []*ScenarioGraphEdge         `json:"edges"`
}

// SerializeScenarioGraph serializes a ScenarioGraph to JSON and writes it to a file
func SerializeScenarioGraph(graph *ScenarioGraph, filePath string) error {
	data := scenarioGraphJSON{
		Vertices: graph.vertices,
		Edges:    graph.edges,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scenario graph: %w", err)
	}

	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write scenario graph to file: %w", err)
	}

	return nil
}

// DeserializeScenarioGraph reads a JSON file and deserializes it into a ScenarioGraph
func DeserializeScenarioGraph(filePath string) (*ScenarioGraph, error) {
	jsonBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario graph file: %w", err)
	}

	var data scenarioGraphJSON
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scenario graph: %w", err)
	}

	graph := &ScenarioGraph{
		vertices: data.Vertices,
		edges:    data.Edges,
	}

	// Ensure maps are initialized if they were nil in JSON
	if graph.vertices == nil {
		graph.vertices = make(map[int]*ScenarioGraphVertex)
	}
	if graph.edges == nil {
		graph.edges = []*ScenarioGraphEdge{}
	}

	return graph, nil
}
