package scenario_builder

import (
	"context"
	"fmt"
	"github.com/d-iii-s/slsbench/internal/model"
	"github.com/d-iii-s/slsbench/internal/service/walker"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"strings"
)

func CreateScenarioFromSpecPath(ctx context.Context, spec *openapi3.T, selector model.PromptSelector) *model.ScenarioGraph {
	dataModel := buildDataModelFromSpec(ctx, spec)
	scenarioGraph := buildScenarioGraph(ctx, dataModel, selector)
	
	// Run topological sort to find order of endpoint calling
	order, err := scenarioGraph.TopologicalSort()
	if err != nil {
		log.Printf("Warning: Failed to compute topological order: %v", err)
		log.Println("Graph may contain cycles. Execution order may be undefined.")
	} else {
		log.Println("Topological order of endpoints:")
		for i, vertexID := range order {
			vertex := scenarioGraph.GetVertex(vertexID)
			if vertex != nil {
				log.Printf("  %d. [%d] %s", i+1, vertexID, vertex.VertexLabel())
			}
		}
	}
	
	return scenarioGraph
}

// BuildDataModelFromSpec builds a DataModel using the walker to traverse the spec.
func buildDataModelFromSpec(ctx context.Context, spec *openapi3.T) *model.DataModel {
	_ = ctx
	dataModel := &model.DataModel{
		Endpoints: make(map[string]*model.Endpoint),
	}

	v := &dataModelVisitor{
		dataModel: dataModel,
	}
	walker.NewWalker(spec).WalkSpec(v)

	return dataModel
}

// dataModelVisitor builds a DataModel while the walker traverses the spec.
type dataModelVisitor struct {
	dataModel *model.DataModel
}

// helper to ensure endpoint and operation exist
func (v *dataModelVisitor) ensureOperation(endpointPath, method string) *model.Operation {
	endpoint, ok := v.dataModel.Endpoints[endpointPath]
	if !ok {
		endpoint = &model.Endpoint{
			Path:       endpointPath,
			Operations: make(map[string]*model.Operation),
		}
		v.dataModel.Endpoints[endpointPath] = endpoint
	}

	op, ok := endpoint.Operations[method]
	if !ok {
		op = &model.Operation{
			Method:     method,
			Parameters: []*model.Field{},
			Responses:  make(map[string]*model.DataStructure),
		}
		endpoint.Operations[method] = op
	}
	return op
}

func parseEndpointAndMethod(path string) (string, string) {
	parts := strings.Split(path, "/")
	methods := map[string]struct{}{
		"GET": {}, "POST": {}, "PUT": {}, "PATCH": {}, "DELETE": {}, "OPTIONS": {}, "HEAD": {}, "TRACE": {},
	}
	for i, p := range parts {
		if _, ok := methods[p]; ok {
			endpoint := "/" + strings.Join(parts[1:i], "/")
			return endpoint, p
		}
	}
	return "", ""
}

func (v *dataModelVisitor) VisitPath(path string, pathItem *openapi3.PathItem) {
	_ = pathItem
	_ = path
}

func (v *dataModelVisitor) VisitOperation(path string, method string, operation *openapi3.Operation) {
	endpointPath, opMethod := parseEndpointAndMethod(path)
	if endpointPath == "" || opMethod == "" {
		return
	}
	_ = operation
	v.ensureOperation(endpointPath, opMethod)
}

func (v *dataModelVisitor) VisitParameter(path string, parameter *openapi3.Parameter) {
	if parameter == nil {
		return
	}
	endpointPath, opMethod := parseEndpointAndMethod(path)
	if endpointPath == "" || opMethod == "" {
		return
	}
	if parameter.Schema == nil || parameter.Schema.Value == nil {
		return
	}
	// Only keep query and path params (mirrors previous behavior)
	if parameter.In != "query" && parameter.In != "path" {
		return
	}

	op := v.ensureOperation(endpointPath, opMethod)

	visitedSchemas := make(map[*openapi3.Schema]struct{})
	field := buildFieldFromSchema(parameter.Name, parameter.Schema, path, visitedSchemas)
	field.Required = parameter.Required
	field.In = parameter.In
	op.Parameters = append(op.Parameters, field)
}

func (v *dataModelVisitor) VisitRequestBody(path string, contentType string, requestBody *openapi3.RequestBody) {
	endpointPath, opMethod := parseEndpointAndMethod(path)
	if endpointPath == "" || opMethod == "" {
		return
	}
	if requestBody == nil || requestBody.Content == nil {
		return
	}
	media := requestBody.Content[contentType]
	if media == nil || media.Schema == nil {
		return
	}

	op := v.ensureOperation(endpointPath, opMethod)
	visitedSchemas := make(map[*openapi3.Schema]struct{})
	op.RequestBody = buildDataStructureFromSchema(path, media.Schema, contentType, visitedSchemas)
}

func (v *dataModelVisitor) VisitResponse(path string, statusCode string, contentType string, response *openapi3.Response) {
	endpointPath, opMethod := parseEndpointAndMethod(path)
	if endpointPath == "" || opMethod == "" {
		return
	}
	if response == nil {
		return
	}

	var ds *model.DataStructure
	if media := response.Content[contentType]; media != nil && media.Schema != nil {
		visitedSchemas := make(map[*openapi3.Schema]struct{})
		ds = buildDataStructureFromSchema(path, media.Schema, contentType, visitedSchemas)
	}

	op := v.ensureOperation(endpointPath, opMethod)
	op.Responses[statusCode] = ds
}

func (v *dataModelVisitor) VisitComponentSchema(name string, schema *openapi3.SchemaRef) {
	_ = name
	_ = schema
}

func (v *dataModelVisitor) VisitProperty(path string, schema *openapi3.SchemaRef) {
	_ = path
	_ = schema
}

func buildScenarioGraph(ctx context.Context, dataModel *model.DataModel, selector model.PromptSelector) *model.ScenarioGraph {
	graph := model.NewScenarioGraph()

	// Build list of available operations from DataModel
	availableOps := buildAvailableOperations(dataModel)

	if len(availableOps) == 0 {
		fmt.Println("No operations found in the data model.")
		return graph
	}

	menuOptions := []string{
		"Add new endpoint (vertex)",
		"Add connection between endpoints",
		"Add field mapping to connection",
		"Done - finish building graph",
	}

	for {
		// Dump current graph state
		fmt.Print(graph.String())

		// Show menu
		menuSelector := model.NewPromptSelector("Select action", menuOptions)
		if selector != nil {
			menuSelector = selector
		}
		idx, _, err := menuSelector.Run()
		if err != nil {
			log.Printf("prompt failed: %v", err)
			break
		}

		switch idx {
		case 0: // Add new endpoint
			addEndpointToGraph(graph, availableOps, selector)
		case 1: // Add connection
			addConnectionToGraph(graph, selector)
		case 2: // Add field mapping
			addFieldMappingToGraph(graph, selector)
		case 3: // Done
			fmt.Println("Finished building scenario graph.")
			fmt.Print(graph.String())
			return graph
		}
	}

	return graph
}

// availableOperation represents an operation that can be added to the graph
type availableOperation struct {
	Path      string
	Method    string
	Operation *model.Operation
}

// Label returns a display label for the operation
func (op *availableOperation) Label() string {
	return fmt.Sprintf("%s %s", op.Method, op.Path)
}

// buildAvailableOperations builds a list of all available operations from DataModel
func buildAvailableOperations(dataModel *model.DataModel) []*availableOperation {
	var ops []*availableOperation
	for _, endpoint := range dataModel.Endpoints {
		for method, operation := range endpoint.Operations {
			ops = append(ops, &availableOperation{
				Path:      endpoint.Path,
				Method:    method,
				Operation: operation,
			})
		}
	}
	return ops
}

// addEndpointToGraph prompts user to select an endpoint and adds it as a vertex
func addEndpointToGraph(graph *model.ScenarioGraph, availableOps []*availableOperation, selector model.PromptSelector) {
	if len(availableOps) == 0 {
		fmt.Println("No available operations to add.")
		return
	}

	// Build options list for endpoint selection (without response codes)
	options := make([]string, len(availableOps))
	for i, op := range availableOps {
		options[i] = op.Label()
	}

	opSelector := model.NewPromptSelector("Select endpoint to add", options)
	if selector != nil {
		opSelector = selector
	}
	idx, _, err := opSelector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}

	selectedOp := availableOps[idx]

	// Now ask for response code
	var responseCode string

	if len(selectedOp.Operation.Responses) == 0 {
		responseCode = "default"
	} else if len(selectedOp.Operation.Responses) == 1 {
		// Only one response, use it directly
		for code := range selectedOp.Operation.Responses {
			responseCode = code
		}
	} else {
		// Multiple responses, ask user to select
		respOptions := make([]string, 0, len(selectedOp.Operation.Responses))
		respCodes := make([]string, 0, len(selectedOp.Operation.Responses))
		for code, resp := range selectedOp.Operation.Responses {
			label := code
			if resp == nil {
				label = code + " (no body)"
			}
			respOptions = append(respOptions, label)
			respCodes = append(respCodes, code)
		}

		respSelector := model.NewPromptSelector("Select response code", respOptions)
		if selector != nil {
			respSelector = selector
		}
		respIdx, _, err := respSelector.Run()
		if err != nil {
			log.Printf("prompt failed: %v", err)
			return
		}
		responseCode = respCodes[respIdx]
	}

	vertex := &model.ScenarioGraphVertex{
		Path:         selectedOp.Path,
		Method:       selectedOp.Method,
		Parameters:   selectedOp.Operation.Parameters,
		RequestBody:  selectedOp.Operation.RequestBody,
		ResponseCode: responseCode,
	}

	id := graph.AddVertex(vertex)
	fmt.Printf("Added vertex [%d]: %s\n", id, vertex.VertexLabel())
}

// addConnectionToGraph prompts user to connect two vertices
func addConnectionToGraph(graph *model.ScenarioGraph, selector model.PromptSelector) {
	vertices := graph.GetVertices()
	if len(vertices) < 2 {
		fmt.Println("Need at least 2 vertices to create a connection.")
		return
	}

	// Build options list for source selection
	srcOptions := make([]string, 0, len(vertices))
	srcIDs := make([]int, 0, len(vertices))
	for id, v := range vertices {
		srcOptions = append(srcOptions, fmt.Sprintf("[%d] %s", id, v.VertexLabel()))
		srcIDs = append(srcIDs, id)
	}

	// Select source vertex
	srcSelector := model.NewPromptSelector("Select SOURCE vertex (response provider)", srcOptions)
	if selector != nil {
		srcSelector = selector
	}
	srcIdx, _, err := srcSelector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}
	srcVertexID := srcIDs[srcIdx]

	// Build options list for target selection (excluding source vertex)
	dstOptions := make([]string, 0, len(vertices)-1)
	dstIDs := make([]int, 0, len(vertices)-1)
	for id, v := range vertices {
		if id == srcVertexID {
			continue // Exclude the selected source vertex
		}
		dstOptions = append(dstOptions, fmt.Sprintf("[%d] %s", id, v.VertexLabel()))
		dstIDs = append(dstIDs, id)
	}

	if len(dstOptions) == 0 {
		fmt.Println("No other vertices available as target.")
		return
	}

	// Select target vertex
	dstSelector := model.NewPromptSelector("Select TARGET vertex (request consumer)", dstOptions)
	if selector != nil {
		dstSelector = selector
	}
	dstIdx, _, err := dstSelector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}
	dstVertexID := dstIDs[dstIdx]

	// Check if edge already exists
	if existingEdge := graph.GetEdge(srcVertexID, dstVertexID); existingEdge != nil {
		fmt.Printf("Connection already exists between [%d] and [%d].\n", srcVertexID, dstVertexID)
		return
	}

	edge := graph.AddEdge(srcVertexID, dstVertexID)
	fmt.Printf("Added connection: [%d] -> [%d]\n", edge.From, edge.To)
}

// addFieldMappingToGraph prompts user to add a field mapping to an existing connection
func addFieldMappingToGraph(graph *model.ScenarioGraph, selector model.PromptSelector) {
	edges := graph.GetEdges()
	if len(edges) == 0 {
		fmt.Println("No connections exist. Create a connection first.")
		return
	}

	// Build options for edge selection
	edgeOptions := make([]string, len(edges))
	for i, e := range edges {
		fromLabel := "(unknown)"
		toLabel := "(unknown)"
		if v := graph.GetVertex(e.From); v != nil {
			fromLabel = v.VertexLabel()
		}
		if v := graph.GetVertex(e.To); v != nil {
			toLabel = v.VertexLabel()
		}
		edgeOptions[i] = fmt.Sprintf("[%d] %s -> [%d] %s", e.From, fromLabel, e.To, toLabel)
	}

	edgeSelector := model.NewPromptSelector("Select connection to add mapping to", edgeOptions)
	if selector != nil {
		edgeSelector = selector
	}
	edgeIdx, _, err := edgeSelector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}

	selectedEdge := edges[edgeIdx]
	srcVertex := graph.GetVertex(selectedEdge.From)
	dstVertex := graph.GetVertex(selectedEdge.To)

	if srcVertex == nil || dstVertex == nil {
		fmt.Println("Invalid edge - vertices not found.")
		return
	}

	// Collect available fields from source vertex (request body, path params, query params)
	sourceFields := collectRequestFields(srcVertex)
	if len(sourceFields) == 0 {
		fmt.Println("No fields available in source vertex (no request body or parameters).")
		return
	}

	// Collect available fields from target vertex (request body, path params, query params)
	targetFields := collectRequestFields(dstVertex)
	if len(targetFields) == 0 {
		fmt.Println("No request fields available in target vertex.")
		return
	}

	// Show existing mappings if any
	if len(selectedEdge.Mappings) > 0 {
		fmt.Println("\nExisting mappings:")
		for src, dst := range selectedEdge.Mappings {
			fmt.Printf("  %s -> %s\n", src, dst)
		}
		fmt.Println()
	}

	// Select source field (from source vertex - request body, path params, query params)
	sourceSelector := model.NewPromptSelector("Select SOURCE field (from previous request)", sourceFields)
	if selector != nil {
		sourceSelector = selector
	}
	srcIdx, _, err := sourceSelector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}
	selectedSourceField := sourceFields[srcIdx]

	// Check if this source field is already mapped and warn user
	if existingTarget, exists := selectedEdge.Mappings[selectedSourceField]; exists {
		fmt.Printf("Warning: Field '%s' is already mapped to '%s'. It will be overwritten.\n", selectedSourceField, existingTarget)
	}

	// Select target field (from target vertex)
	targetSelector := model.NewPromptSelector("Select TARGET field (to map to)", targetFields)
	if selector != nil {
		targetSelector = selector
	}
	dstIdx, _, err := targetSelector.Run()
	if err != nil {
		log.Printf("prompt failed: %v", err)
		return
	}
	selectedTargetField := targetFields[dstIdx]

	// Add the mapping
	selectedEdge.AddMapping(selectedSourceField, selectedTargetField)
	fmt.Printf("Added mapping: %s -> %s\n", selectedSourceField, selectedTargetField)
}

// collectRequestFields collects all field paths from a vertex's request (body, params)
func collectRequestFields(vertex *model.ScenarioGraphVertex) []string {
	var fields []string

	// Collect body fields
	if vertex.RequestBody != nil {
		bodyFields := collectFieldPaths(vertex.RequestBody.Fields, "body")
		fields = append(fields, bodyFields...)
	}

	// Collect parameters (query and path)
	for _, param := range vertex.Parameters {
		if param == nil {
			continue
		}
		location := "param"
		if param.In != "" {
			location = param.In
		}
		fields = append(fields, fmt.Sprintf("%s.%s", location, param.Name))
	}

	return fields
}

// collectFieldPaths recursively collects all field paths from fields
func collectFieldPaths(fields []*model.Field, prefix string) []string {
	var paths []string
	for _, field := range fields {
		if field == nil {
			continue
		}
		fieldPath := prefix + "." + field.Name

		// Add this field
		paths = append(paths, fieldPath)

		// Recursively add nested properties
		if field.Properties != nil {
			for _, prop := range field.Properties {
				nestedPaths := collectFieldPaths([]*model.Field{prop}, fieldPath)
				paths = append(paths, nestedPaths...)
			}
		}

		// Handle array items
		if field.Items != nil {
			itemPaths := collectFieldPaths([]*model.Field{field.Items}, fieldPath+"[]")
			paths = append(paths, itemPaths...)
		}
	}
	return paths
}

// buildDataModel builds a tree structure containing endpoints, operations, and data structures with fields.
func buildDataModel(ctx context.Context, spec *openapi3.T) *model.DataModel {
	dataModel := &model.DataModel{
		Endpoints: make(map[string]*model.Endpoint),
	}

	// Build the tree by iterating through all paths and operations
	for pathName, pathItem := range spec.Paths.Map() {
		endpoint := &model.Endpoint{
			Path:       pathName,
			Operations: make(map[string]*model.Operation),
		}

		for operationName, operationsItem := range pathItem.Operations() {
			operation := &model.Operation{
				Method:     operationName,
				Parameters: []*model.Field{},
				Responses:  make(map[string]*model.DataStructure),
			}

			// Extract parameters (query and path)
			// Create a new visited map for each parameter to allow same schema in different contexts
			for _, paramRef := range operationsItem.Parameters {
				if paramRef == nil || paramRef.Value == nil {
					continue
				}
				param := paramRef.Value
				if param.In == "query" || param.In == "path" {
					if param.Schema != nil && param.Schema.Value != nil {
						visitedSchemas := make(map[*openapi3.Schema]struct{})
						field := buildFieldFromSchema(param.Name, param.Schema, fmt.Sprintf("%s/%s/parameter/%s", pathName, operationName, param.Name), visitedSchemas)
						field.Required = param.Required
						field.In = param.In // Store parameter location (query or path)
						operation.Parameters = append(operation.Parameters, field)
					}
				}
			}

			// Extract request body
			if operationsItem.RequestBody != nil && operationsItem.RequestBody.Value != nil {
				requestBody := operationsItem.RequestBody.Value
				for contentType, mediaType := range requestBody.Content {
					if mediaType != nil && mediaType.Schema != nil {
						// Create a new visited map for each request body to allow same schema in different contexts
						visitedSchemas := make(map[*openapi3.Schema]struct{})
						ds := buildDataStructureFromSchema(
							fmt.Sprintf("%s/%s/requestBody/%s", pathName, operationName, contentType),
							mediaType.Schema,
							contentType,
							visitedSchemas,
						)
						operation.RequestBody = ds
					}
				}
			}

			// Extract responses (including those without body)
			for statusCode, responseRef := range operationsItem.Responses.Map() {
				if responseRef == nil || responseRef.Value == nil {
					continue
				}
				response := responseRef.Value

				// Check if response has content/body
				hasContent := false
				for contentType, mediaType := range response.Content {
					if mediaType != nil && mediaType.Schema != nil {
						hasContent = true
						// Create a new visited map for each response to allow same schema in different contexts
						visitedSchemas := make(map[*openapi3.Schema]struct{})
						ds := buildDataStructureFromSchema(
							fmt.Sprintf("%s/%s/response/%s/%s", pathName, operationName, statusCode, contentType),
							mediaType.Schema,
							contentType,
							visitedSchemas,
						)
						operation.Responses[statusCode] = ds
					}
				}

				// Include response codes without body (e.g., 204 No Content)
				if !hasContent {
					operation.Responses[statusCode] = nil
				}
			}

			endpoint.Operations[operationName] = operation
		}

		dataModel.Endpoints[pathName] = endpoint
	}

	return dataModel
}

// buildDataStructureFromSchema builds a DataStructure from a schema reference
func buildDataStructureFromSchema(path string, schemaRef *openapi3.SchemaRef, contentType string, visitedSchemas map[*openapi3.Schema]struct{}) *model.DataStructure {
	ds := &model.DataStructure{
		Name:        path,
		ContentType: contentType,
		Fields:      []*model.Field{},
	}

	if schemaRef == nil {
		return ds
	}

	if schemaRef.Ref != "" {
		ds.Ref = schemaRef.Ref
	}

	if schemaRef.Value == nil {
		return ds
	}

	// Build fields from the schema
	fields := buildFieldsFromSchema(schemaRef.Value, path, visitedSchemas)
	ds.Fields = fields

	return ds
}

// buildFieldsFromSchema recursively builds fields from a schema
func buildFieldsFromSchema(schema *openapi3.Schema, basePath string, visitedSchemas map[*openapi3.Schema]struct{}) []*model.Field {
	if schema == nil {
		return []*model.Field{}
	}

	// Avoid infinite loops
	if _, seen := visitedSchemas[schema]; seen {
		return []*model.Field{}
	}
	visitedSchemas[schema] = struct{}{}
	defer delete(visitedSchemas, schema)

	fields := []*model.Field{}

	// Handle object properties
	for propName, propRef := range schema.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}

		fieldPath := fmt.Sprintf("%s.%s", basePath, propName)
		field := buildFieldFromSchema(propName, propRef, fieldPath, visitedSchemas)

		// Check if field is required
		for _, required := range schema.Required {
			if required == propName {
				field.Required = true
				break
			}
		}

		fields = append(fields, field)
	}

	// Handle allOf - merge fields from all schemas in allOf
	for i, allOfRef := range schema.AllOf {
		if allOfRef != nil && allOfRef.Value != nil {
			allOfPath := fmt.Sprintf("%s/allOf[%d]", basePath, i)
			allOfFields := buildFieldsFromSchema(allOfRef.Value, allOfPath, visitedSchemas)
			fields = append(fields, allOfFields...)
		}
	}

	return fields
}

// buildFieldFromSchema builds a Field from a schema reference
func buildFieldFromSchema(name string, schemaRef *openapi3.SchemaRef, path string, visitedSchemas map[*openapi3.Schema]struct{}) *model.Field {
	field := &model.Field{
		Name:   name,
		Path:   path,
		Schema: nil,
	}

	if schemaRef == nil {
		return field
	}

	if schemaRef.Ref != "" {
		field.Ref = schemaRef.Ref
	}

	if schemaRef.Value == nil {
		return field
	}

	schema := schemaRef.Value
	field.Schema = schema

	// Extract type
	if schema.Type != nil && len(*schema.Type) > 0 {
		field.Type = (*schema.Type)[0]
	}
	field.Format = schema.Format

	// Extract extensions (hint and unique)
	if schema.Extensions != nil {
		// Extract x-user-hint (set by enrich service)
		if hint, ok := schema.Extensions["x-user-hint"].(string); ok {
			field.Hint = hint
		}
		// Extract x-slsbench-unique (set by enrich service)
		if unique, ok := schema.Extensions["x-slsbench-unique"].(bool); ok {
			field.Unique = unique
		}
	}

	// Extract numeric constraints (minimum, maximum)
	field.Min = schema.Min
	field.Max = schema.Max

	// Extract string length constraints (minLength, maxLength)
	field.MinLength = schema.MinLength
	field.MaxLength = schema.MaxLength

	// Extract pattern (regex) for string validation
	field.Pattern = schema.Pattern

	// Handle array items
	if schema.Items != nil {
		itemPath := path + "[]"
		field.Items = buildFieldFromSchema("", schema.Items, itemPath, visitedSchemas)
	}

	// Handle object properties (nested objects)
	if len(schema.Properties) > 0 {
		field.Properties = make(map[string]*model.Field)
		for propName, propRef := range schema.Properties {
			if propRef != nil && propRef.Value != nil {
				propPath := fmt.Sprintf("%s.%s", path, propName)
				field.Properties[propName] = buildFieldFromSchema(propName, propRef, propPath, visitedSchemas)
			}
		}
	}
	return field
}
