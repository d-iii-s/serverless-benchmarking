// Package flowgen parses a DSL YAML file describing benchmark flows,
// computes the number of request bodies needed per endpoint using
// wrk2 parameters and Weighted Round Robin (WRR), and delegates
// body generation to the datagen service.
package flowgen

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// DSL types
// ---------------------------------------------------------------------------

// DSL is the top-level structure of the benchmark DSL file.
type DSL struct {
	Stages map[string]Stage `yaml:"stages"`
}

// Stage describes a single benchmark stage.
type Stage struct {
	Wrk2Params string     `yaml:"wrk2params"`
	Flow       []FlowNode `yaml:"flow"`
}

// FlowNode is one node in a stage flow.
type FlowNode struct {
	Name        string `yaml:"-"` // populated during parsing from the YAML key
	OperationID string `yaml:"operationId"`
	Endpoint    string `yaml:"endpoint"`
	Method      string `yaml:"method"`
	EntryNode   bool   `yaml:"entrynode"`
	Edges       []Edge `yaml:"edges"`
}

// Edge is a weighted outgoing edge from one flow node to another.
type Edge struct {
	To       string    `yaml:"to"`
	Weight   float64   `yaml:"weight"`
	Mappings []Mapping `yaml:"mappings"`
}

// Mapping describes a field mapping between source and destination.
type Mapping struct {
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

// NodeBodyCount holds the computed body count for a single node.
type NodeBodyCount struct {
	NodeName string
	Endpoint string
	Method   string
	Count    int
}

// ---------------------------------------------------------------------------
// DSL parsing
// ---------------------------------------------------------------------------

// ParseDSL reads and parses a DSL YAML file into a DSL struct.
func ParseDSL(path string) (*DSL, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read DSL file %q: %w", path, err)
	}

	// The DSL YAML uses a pattern where each flow list item has a node
	// name as a key alongside endpoint/method/etc.  Because of this
	// mixed-key structure we first unmarshal into raw form, then convert.
	var raw struct {
		Stages map[string]struct {
			Wrk2Params string      `yaml:"wrk2params"`
			Flow       []yaml.Node `yaml:"flow"`
		} `yaml:"stages"`
	}

	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse DSL YAML: %w", err)
	}

	dsl := &DSL{Stages: make(map[string]Stage, len(raw.Stages))}

	for stageName, rawStage := range raw.Stages {
		stage := Stage{Wrk2Params: rawStage.Wrk2Params}

		for _, node := range rawStage.Flow {
			fn, err := parseFlowNode(&node)
			if err != nil {
				return nil, fmt.Errorf("stage %q: %w", stageName, err)
			}
			stage.Flow = append(stage.Flow, fn)
		}

		dsl.Stages[stageName] = stage
	}

	return dsl, nil
}

// parseFlowNode converts a YAML mapping node into a FlowNode.
// The YAML structure looks like:
//
//   - node1:
//     operationId: createUser
//     ...
//
// The first key that is not a known property is treated as the node name.
func parseFlowNode(n *yaml.Node) (FlowNode, error) {
	if n.Kind != yaml.MappingNode {
		return FlowNode{}, fmt.Errorf("expected mapping node, got %v", n.Kind)
	}

	fn := FlowNode{}

	for i := 0; i < len(n.Content)-1; i += 2 {
		key := n.Content[i].Value
		val := n.Content[i+1]

		switch key {
		case "operationId":
			fn.OperationID = val.Value
		case "endpoint":
			fn.Endpoint = val.Value
		case "method":
			fn.Method = val.Value
		case "entrynode":
			fn.EntryNode = val.Value == "true"
		case "edges":
			var edges []Edge
			if err := val.Decode(&edges); err != nil {
				return FlowNode{}, fmt.Errorf("failed to decode edges: %w", err)
			}
			fn.Edges = edges
		default:
			// First unknown key is treated as the node name.
			if fn.Name == "" {
				fn.Name = key
			}
		}
	}

	if fn.Name == "" {
		fn.Name = fn.OperationID // fallback
	}
	if fn.OperationID == "" {
		return FlowNode{}, fmt.Errorf("missing required operationId")
	}

	return fn, nil
}

// ---------------------------------------------------------------------------
// wrk2 parameter parsing
// ---------------------------------------------------------------------------

var (
	rateRe     = regexp.MustCompile(`-R\s*(\d+)`)
	durationRe = regexp.MustCompile(`-d\s*(\d+)([smhSMH]?)`)
)

// Wrk2Config holds the parsed wrk2 rate and duration.
type Wrk2Config struct {
	Rate     int // requests per second
	Duration int // seconds
}

// ParseWrk2Params extracts the rate (-R) and duration (-d) from a wrk2
// parameter string.  Duration suffixes: s (default), m, h.
func ParseWrk2Params(params string) (Wrk2Config, error) {
	cfg := Wrk2Config{}

	if m := rateRe.FindStringSubmatch(params); m != nil {
		r, err := strconv.Atoi(m[1])
		if err != nil {
			return cfg, fmt.Errorf("invalid rate %q: %w", m[1], err)
		}
		cfg.Rate = r
	} else {
		return cfg, fmt.Errorf("missing -R (rate) in wrk2params %q", params)
	}

	if m := durationRe.FindStringSubmatch(params); m != nil {
		d, err := strconv.Atoi(m[1])
		if err != nil {
			return cfg, fmt.Errorf("invalid duration %q: %w", m[1], err)
		}
		switch strings.ToLower(m[2]) {
		case "m":
			d *= 60
		case "h":
			d *= 3600
		default: // "s" or empty
		}
		cfg.Duration = d
	} else {
		return cfg, fmt.Errorf("missing -d (duration) in wrk2params %q", params)
	}

	return cfg, nil
}

// TotalRequests returns Rate * Duration.
func (c Wrk2Config) TotalRequests() int {
	return c.Rate * c.Duration
}

// ---------------------------------------------------------------------------
// Weighted Round Robin body count computation
// ---------------------------------------------------------------------------

// ComputeBodyCounts walks the flow graph starting from entry nodes and
// distributes the total request count according to edge weights.
// Only nodes whose HTTP method typically carries a request body
// (POST, PUT, PATCH) will have a non-zero count.
func ComputeBodyCounts(stage Stage, totalRequests int) ([]NodeBodyCount, error) {
	// Build a name -> FlowNode index.
	nodeByName := make(map[string]*FlowNode, len(stage.Flow))
	for i := range stage.Flow {
		nodeByName[stage.Flow[i].Name] = &stage.Flow[i]
	}

	// Accumulate raw request counts per node name.
	counts := make(map[string]int, len(stage.Flow))

	// Find entry nodes.
	var entries []*FlowNode
	for i := range stage.Flow {
		if stage.Flow[i].EntryNode {
			entries = append(entries, &stage.Flow[i])
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no entry node found in flow")
	}

	// BFS / DFS propagation using a simple queue.
	type work struct {
		name  string
		count int
	}
	queue := make([]work, 0, len(stage.Flow))

	// Divide total evenly among entry nodes (usually just one).
	perEntry := totalRequests / len(entries)
	for _, e := range entries {
		queue = append(queue, work{name: e.Name, count: perEntry})
	}

	for len(queue) > 0 {
		w := queue[0]
		queue = queue[1:]

		counts[w.name] += w.count

		node, ok := nodeByName[w.name]
		if !ok || len(node.Edges) == 0 {
			continue
		}

		for _, edge := range node.Edges {
			childCount := int(math.Round(float64(w.count) * edge.Weight))
			queue = append(queue, work{name: edge.To, count: childCount})
		}
	}

	// Build result, only including body-bearing methods.
	result := make([]NodeBodyCount, 0, len(stage.Flow))
	for _, fn := range stage.Flow {
		c := counts[fn.Name]
		if !methodHasBody(fn.Method) {
			c = 0
		}
		result = append(result, NodeBodyCount{
			NodeName: fn.Name,
			Endpoint: fn.Endpoint,
			Method:   fn.Method,
			Count:    c,
		})
	}

	return result, nil
}

// methodHasBody returns true for HTTP methods that carry a request body.
func methodHasBody(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH":
		return true
	default:
		return false
	}
}
