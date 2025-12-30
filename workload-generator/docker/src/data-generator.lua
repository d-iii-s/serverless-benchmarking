local json = require("json")
local Faker = require("faker")

-- Module table
local data_generator = {}

-- Cached scenario data
local cached_scenario = nil

-------------------------------------------------------
-- Fake data generators based on hints
-------------------------------------------------------
-- Non-symmetric pairing function for two non-negative numbers
-- pair(x, y) != pair(y, x) when x != y
-- Formula: pair(x, y) = x * (x + y + 1) + y
function pair(x, y)
    return x * (x + y + 1) + y
end

-- todo: delete question about uniqness - by deafault all unique
local function generate_fake_data(hint, field_type, format, thread_id, conn_id, unique_values_generated, min, max, pattern, minLength, maxLength)
    -- Generate a more unique seed by combining multiple entropy sources
    local salt = math.random(1, 2147483647)
    
    -- Add hint to seed for additional uniqueness per field type
    local hint_hash = 0
    if hint then
        for i = 1, #hint do
            hint_hash = hint_hash * 31 + string.byte(hint, i)
        end
    end
    
    -- Combine all sources using multiple pairing functions for better distribution
    -- This creates a more unique seed by mixing thread, connection, counter, salt, and hint
    -- Using nested pairs ensures good distribution and uniqueness
    local seed = pair(
        pair(pair(thread_id, conn_id), unique_values_generated),
        pair(salt, hint_hash % 2147483647)
    )
    
    -- Create faker instance with seed for reproducible results
    local faker = Faker:new({seed = seed})

    -- Name hints
    if hint == "firstName" or hint == "first_name" or hint == "name" then
        return faker:name()
    elseif hint == "lastName" or hint == "last_name" or hint == "surname" then
        return faker:name()
    elseif hint == "fullName" or hint == "full_name" then
        return faker:name()
    -- Internet hints
    elseif hint == "email" then
        return faker:email()
    elseif hint == "username" or hint == "userName" then
        local opts = {}
        if pattern then opts.pattern = pattern end
        if minLength ~= nil then opts.minLength = minLength end
        if maxLength ~= nil then opts.maxLength = maxLength end
        if next(opts) then
            return faker:string(opts)
        else
            return faker:string()
        end
    elseif hint == "password" then
        local opts = {}
        if minLength ~= nil then opts.minLength = minLength
        elseif min ~= nil then opts.minLength = min end
        if maxLength ~= nil then opts.maxLength = maxLength
        elseif max ~= nil then opts.maxLength = max end
        if next(opts) then
            return faker:password(opts)
        else
            return faker:password()
        end
    elseif hint == "url" then
        return faker:url()
    elseif hint == "domainName" or hint == "domain_name" or hint == "hostname" then
        return faker:hostname()
    elseif hint == "ip" or hint == "ipAddress" or hint == "ip_address" then
        return faker:ipv4()
    elseif hint == "ipv4" or hint == "ipv4Address" or hint == "ipv4_address" then
        return faker:ipv4()
    elseif hint == "ipv6" or hint == "ipv6Address" or hint == "ipv6_address" then
        return faker:ipv6()
    elseif hint == "uri" then
        return faker:uri()
    -- Address hints
    elseif hint == "city" then
        return faker:city()
    elseif hint == "state" or hint == "stateAbbr" or hint == "state_abbr" then
        return faker:state()
    elseif hint == "country" then
        return faker:country()
    -- String hints
    elseif hint == "word" or hint == "string" then
        local opts = {}
        if pattern then opts.pattern = pattern end
        if format then opts.format = format end
        if minLength ~= nil then opts.minLength = minLength end
        if maxLength ~= nil then opts.maxLength = maxLength end
        if next(opts) then
            return faker:string(opts)
        else
            return faker:string()
        end
    -- Date hints
    elseif hint == "date" or format == "date" then
        if min ~= nil or max ~= nil then
            return faker:date({min = min, max = max})
        else
            return faker:date()
        end
    elseif hint == "timestamp" or hint == "iso8601" or format == "date-time" or hint == "dateTime" or hint == "date_time" then
        if min ~= nil or max ~= nil then
            return tostring(faker:timestamp({min = min, max = max}))
        else
            return tostring(faker:timestamp())
        end
    elseif hint == "dateTime" or hint == "date_time" then
        if min ~= nil or max ~= nil then
            return faker:dateTime({min = min, max = max})
        else
            return faker:dateTime()
        end
    -- Datatype hints
    elseif hint == "uuid" then
        return faker:uuid()
    elseif hint == "number" or hint == "int" or hint == "integer" or field_type == "integer" then
        if min ~= nil or max ~= nil then
            return faker:integer({min = min, max = max})
        else
            return faker:integer()
        end
    elseif hint == "float" or hint == "double" or field_type == "number" then
        if min ~= nil or max ~= nil then
            return faker:integer({min = min, max = max}) / 100.0
        else
            return faker:integer() / 100.0
        end
    elseif hint == "boolean" or hint == "bool" or field_type == "boolean" then
        return faker:boolean()
    elseif hint == "byte" or hint == "bytes" then
        local opts = {}
        if minLength ~= nil then opts.minLength = minLength end
        if maxLength ~= nil then opts.maxLength = maxLength end
        if next(opts) then
            return faker:byte(opts)
        else
            return faker:byte()
        end
    elseif hint == "binary" then
        local opts = {}
        if minLength ~= nil then opts.minLength = minLength end
        if maxLength ~= nil then opts.maxLength = maxLength end
        if next(opts) then
            return faker:binary(opts)
        else
            return faker:binary()
        end
    elseif hint == "id" then
        local opts = {}
        if minLength ~= nil then opts.minLength = minLength end
        if maxLength ~= nil then opts.maxLength = maxLength end
        if next(opts) then
            return faker:id(opts)
        else
            return faker:id()
        end
    else
        -- Default fallback based on field_type
        if field_type == "string" then
            local opts = {}
            if pattern then opts.pattern = pattern end
            if format then opts.format = format end
            if minLength ~= nil then opts.minLength = minLength end
            if maxLength ~= nil then opts.maxLength = maxLength end
            if next(opts) then
                return faker:string(opts)
            else
                return faker:string() .. "_" .. seed
            end
        elseif field_type == "integer" or field_type == "number" then
            if min ~= nil or max ~= nil then
                return faker:integer({min = min, max = max})
            else
                return faker:integer()
            end
        elseif field_type == "boolean" then
            return faker:boolean()
        else
            local opts = {}
            if pattern then opts.pattern = pattern end
            if format then opts.format = format end
            if minLength ~= nil then opts.minLength = minLength end
            if maxLength ~= nil then opts.maxLength = maxLength end
            if next(opts) then
                return faker:string(opts)
            else
                return faker:string() .. "_" .. seed
            end
        end
    end
end

-------------------------------------------------------
-- URL encoding helper
-------------------------------------------------------

local function url_encode(str)
    if str == nil then return "" end
    str = tostring(str)
    str = string.gsub(str, "([^%w%-_.~])", function(c)
        return string.format("%%%02X", string.byte(c))
    end)
    return str
end

-------------------------------------------------------
-- Load scenario
-------------------------------------------------------

function data_generator.load_scenario(scenario_path)
    scenario_path = scenario_path or "scenario.json"

    local file = io.open(scenario_path, "r")
    if not file then
        error("Failed to open scenario file: " .. scenario_path)
    end

    local content = file:read("*all")
    file:close()

    cached_scenario = json.decode(content)
    return cached_scenario
end

function data_generator.get_scenario(scenario_path)
    if cached_scenario then
        return cached_scenario
    end
    return data_generator.load_scenario(scenario_path)
end

-------------------------------------------------------
-- Get vertex count
-------------------------------------------------------

function data_generator.get_vertex_count(scenario_path)
    local scenario = data_generator.get_scenario(scenario_path)

    local count = 0
    if scenario.vertices then
        for _ in pairs(scenario.vertices) do
            count = count + 1
        end
    end

    return count
end

-------------------------------------------------------
-- Find mappings from edges for a given target vertex
-- Returns a table: { ["path.cid"] = { from_vertex = 1, source_type = "body|path|query|response", source_field = "username" }, ... }
-------------------------------------------------------

local function find_mappings_for_vertex(scenario, target_vertex)
    local mappings = {}

    if not scenario.edges then
        return mappings
    end

    for _, edge in ipairs(scenario.edges) do
        if edge.to == target_vertex then
            for source, dest in pairs(edge.mappings) do
                -- source can be like "body.username", "path.param", "query.param", or "response.id"
                -- dest is like "path.cid", "body.username", etc.
                local response_field = source:match("^response%.(.+)$")
                local body_field = source:match("^body%.(.+)$")
                local path_field = source:match("^path%.(.+)$")
                local query_field = source:match("^query%.(.+)$")
                
                if response_field then
                    mappings[dest] = {
                        from_vertex = edge.from,
                        source_type = "response",
                        source_field = response_field
                    }
                elseif body_field then
                    mappings[dest] = {
                        from_vertex = edge.from,
                        source_type = "body",
                        source_field = body_field
                    }
                elseif path_field then
                    mappings[dest] = {
                        from_vertex = edge.from,
                        source_type = "path",
                        source_field = path_field
                    }
                elseif query_field then
                    mappings[dest] = {
                        from_vertex = edge.from,
                        source_type = "query",
                        source_field = query_field
                    }
                end
            end
        end
    end

    return mappings
end

-------------------------------------------------------
-- Extract value from a parsed response using field path
-------------------------------------------------------

local function get_response_value(response, field_path)
    if not response then return nil end

    -- Handle simple field access like "id"
    local value = response[field_path]
    if value ~= nil then
        return value
    end

    -- Handle nested paths like "nested.field"
    local current = response
    for part in field_path:gmatch("[^%.]+") do
        if type(current) ~= "table" then
            return nil
        end
        current = current[part]
        if current == nil then
            return nil
        end
    end

    return current
end

-------------------------------------------------------
-- Build request for a given vertex
-- Parameters:
--   scenario: parsed scenario JSON (or nil to use cached)
--   responses: table of parsed responses keyed by vertex index (number or string)
--   generated_values: table of generated values from previous requests keyed by vertex index
--   next_step: the vertex index to build request for
--   thread_id: thread identifier for unique data generation
--   conn_id: connection identifier for unique data generation
--   unique_values_generated: counter for unique values
-- Returns: path (string), method (string), body (string), content_type (string), unique_values_generated (number), generated_values (table)
-------------------------------------------------------

function data_generator.build_request(scenario, responses, previous_generated_values, next_step, thread_id, conn_id,
    unique_values_generated)
    scenario = scenario or data_generator.get_scenario()
    responses = responses or {}
    previous_generated_values = previous_generated_values or {}

    local vertex_key = tostring(next_step)
    local vertex = scenario.vertices[vertex_key]

    if not vertex then
        error("Vertex not found: " .. vertex_key)
    end

    -- Find all mappings from edges pointing to this vertex
    local edge_mappings = find_mappings_for_vertex(scenario, next_step)

    -- Table to store all generated values as a simple field-to-value map
    local generated_values = {}

    -- Build parameter values (for path and query)
    local param_values = {}
    local query_params = {}

    if vertex.parameters then
        for _, param in ipairs(vertex.parameters) do
            local param_name = param.name
            local mapping_key = (param["in"] or "path") .. "." .. param_name
            local value = nil

            -- Check if there's a mapping from a previous response or generated_values
            local mapping = edge_mappings[mapping_key]
            if mapping then
                if mapping.source_type == "response" then
                    local from_response = responses[mapping.from_vertex] or responses[tostring(mapping.from_vertex)]
                    if from_response then
                        value = get_response_value(from_response, mapping.source_field)
                    end
                else
                    -- Look up in generated_values from previous request
                    local from_generated = previous_generated_values[mapping.from_vertex] or previous_generated_values[tostring(mapping.from_vertex)]
                    if from_generated then
                        local source_key = mapping.source_type .. "." .. mapping.source_field
                        value = from_generated[source_key]
                    end
                end
            end

            -- If no mapping found, generate fake data
            if value == nil then
                value = generate_fake_data(param.hint, param.type, param.format, thread_id, conn_id, unique_values_generated, param.min, param.max, param.pattern, param.minLength, param.maxLength)
                unique_values_generated = unique_values_generated + 1
            end

            if param["in"] == "query" then
                query_params[param_name] = value
                generated_values["query." .. param_name] = value
            else
                param_values[param_name] = value
                generated_values["path." .. param_name] = value
            end
        end
    end

    -- Build the path with substitutions
    local path = vertex.path
    for param_name, param_value in pairs(param_values) do
        path = path:gsub("{" .. param_name .. "}", tostring(param_value))
    end

    -- Append query parameters if any
    if next(query_params) then
        local query_parts = {}
        for k, v in pairs(query_params) do
            table.insert(query_parts, url_encode(k) .. "=" .. url_encode(v))
        end
        path = path .. "?" .. table.concat(query_parts, "&")
    end

    -- Build request body
    local body = ""
    local content_type = ""

    if vertex.requestBody and vertex.requestBody.fields then
        content_type = vertex.requestBody.contentType or "application/x-www-form-urlencoded"
        local body_fields = {}

        for _, field in ipairs(vertex.requestBody.fields) do
            local field_name = field.name
            local mapping_key = "body." .. field_name
            local value = nil

            -- Check if there's a mapping from a previous response or generated_values
            local mapping = edge_mappings[mapping_key]
            if mapping then
                if mapping.source_type == "response" then
                    local from_response = responses[mapping.from_vertex] or responses[tostring(mapping.from_vertex)]
                    if from_response then
                        value = get_response_value(from_response, mapping.source_field)
                    end
                else
                    -- Look up in generated_values from previous request
                    local from_generated = previous_generated_values[mapping.from_vertex] or previous_generated_values[tostring(mapping.from_vertex)]
                    if from_generated then
                        local source_key = mapping.source_type .. "." .. mapping.source_field
                        value = from_generated[source_key]
                    end
                end
            end

            -- If no mapping found, generate fake data
            if value == nil then
                value = generate_fake_data(field.hint, field.type, field.format, thread_id, conn_id, unique_values_generated, field.min, field.max, field.pattern, field.minLength, field.maxLength)
                unique_values_generated = unique_values_generated + 1
            end

            body_fields[field_name] = value
            generated_values["body." .. field_name] = value
        end

        -- Encode body based on content type
        if content_type == "application/x-www-form-urlencoded" then
            local parts = {}
            for k, v in pairs(body_fields) do
                table.insert(parts, url_encode(k) .. "=" .. url_encode(v))
            end
            body = table.concat(parts, "&")
        elseif content_type == "application/json" then
            body = json.encode(body_fields)
        else
            -- Fallback to form-urlencoded
            local parts = {}
            for k, v in pairs(body_fields) do
                table.insert(parts, url_encode(k) .. "=" .. url_encode(v))
            end
            body = table.concat(parts, "&")
        end
    end

    return path, vertex.method, body, content_type, unique_values_generated, generated_values
end


-- Export generate_fake_data for testing
data_generator.generate_fake_data = generate_fake_data

-- Return the module table
return data_generator