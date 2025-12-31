#!/usr/bin/env lua

-- Script to measure time to first successful request
-- Uses SERVICE_NAME and PORT environment variables
-- Queries the first endpoint from scenario.json until successful

-- Add path to find modules (current directory first)
package.path = "./?.lua;" .. package.path

-- Load required modules
local json = require("json")
local data_generator = require("data-generator")

-- Load luaposix modules
local posix = require("posix")
local posix_stat = require("posix.sys.stat")

-- Load luasocket modules
local socket = require("socket")
local http = require("socket.http")
local ltn12 = require("ltn12")

-------------------------------------------------------
-- Helper Functions
-------------------------------------------------------

-- Create directory recursively (like mkdir -p)
local function mkdir_p(path)
    local parts = {}
    for part in path:gmatch("[^/]+") do
        table.insert(parts, part)
    end
    
    local current_path = ""
    for i, part in ipairs(parts) do
        if i == 1 and path:sub(1, 1) == "/" then
            current_path = "/" .. part
        else
            current_path = current_path .. (current_path == "" and "" or "/") .. part
        end
        
        local ok, err = posix_stat.mkdir(current_path, "0755")
        if not ok and err ~= "EEXIST" then
            return false, err
        end
    end
    return true
end

-------------------------------------------------------
-- HTTP Request Functions
-------------------------------------------------------

-- Make HTTP request using LuaSocket
local function make_request(method, url, headers, body, timeout)
    timeout = timeout or 0.1  -- Default timeout of 100ms
    
    local start_time = socket.gettime()
    local response_body = {}
    
    local request = {
        method = method,
        url = url,
        headers = headers or {},
        source = body and ltn12.source.string(body) or nil,
        sink = ltn12.sink.table(response_body),
        timeout = timeout
    }
    
    local result, status_code, response_headers, status_line = http.request(request)
    local end_time = socket.gettime()
    local request_time = end_time - start_time
    
    if result then
        local body_text = table.concat(response_body)
        return status_code, response_headers, body_text, request_time
    else
        -- Timeout or connection error
        return 0, {}, nil, request_time
    end
end

-------------------------------------------------------
-- Main Function
-------------------------------------------------------

local function main()
    -- Get environment variables
    local service_name = os.getenv("SERVICE_NAME")
    local port = os.getenv("PORT")
    
    if not service_name or not port then
        error("SERVICE_NAME and PORT environment variables must be set")
    end
    
    local base_url = string.format("http://%s:%s", service_name, port)
    print(string.format("Target service: %s", base_url))
    
    -- Load scenario
    local scenario_path = "scenario.json"
    print(string.format("Loading scenario from: %s", scenario_path))
    local scenario = data_generator.load_scenario(scenario_path)
    
    if not scenario or not scenario.vertices then
        error("Failed to load scenario or scenario has no vertices")
    end
    
    -- Get first endpoint (vertex "0")
    local first_vertex_key = "0"
    local first_vertex = scenario.vertices[first_vertex_key]
    
    if not first_vertex then
        error("First vertex (0) not found in scenario")
    end
    
    print(string.format("\nFirst endpoint: %s %s", first_vertex.method, first_vertex.path))
    
    -- Generate dummy data for the first endpoint
    local unique_seed = os.time() * 1000 + math.random(1, 999)
    local thread_id = unique_seed % 1000000
    local conn_id = (unique_seed + 1) % 1000000
    local unique_values_generated = math.random(1, 10000)
    
    local path, method, body, content_type, _, generated_values = data_generator.build_request(
        scenario,
        {},  -- No previous responses
        {},  -- No previous generated values
        0,   -- First vertex
        thread_id,
        conn_id,
        unique_values_generated
    )
    
    local full_url = base_url .. path
    print(string.format("Full URL: %s", full_url))
    print(string.format("Method: %s", method))
    if body and body ~= "" then
        print(string.format("Body: %s", body))
    end
    print(string.format("Content-Type: %s", content_type))
    
    -- Prepare headers
    local headers = {
        ["Content-Type"] = content_type,
        ["Accept"] = "application/json"
    }
    
    -- Measure time to first successful request
    print("\nStarting requests until first successful response...")
    print("(A successful request is one with status code 200-299)\n")
    
    -- Get start time using luaposix gettimeofday for high precision
    local start_tv = posix.gettimeofday()
    local start_time = start_tv.sec + start_tv.usec / 1000000.0
    
    local attempt = 0
    local success = false
    local final_status = nil
    local final_body = nil
    local first_success_timestamp = nil
    local request_timeout = 0.1  -- 100ms timeout per request
    
    while not success do
        attempt = attempt + 1
        
        -- Make synchronous request
        local status, response_headers, response_body, request_time = make_request(method, full_url, headers, body, request_timeout)
        
        if status then
            if status == 0 then
                print(string.format("Attempt %d: Timeout (%.3f seconds)", attempt, request_time))
            else
                print(string.format("Attempt %d: Status %d (%.3f seconds)", attempt, status, request_time))
                
                -- Consider 2xx status codes as successful
                if status >= 200 and status < 300 then
                    success = true
                    final_status = status
                    final_body = response_body
                    
                    -- Get timestamp of first success
                    local success_tv = posix.gettimeofday()
                    first_success_timestamp = success_tv.sec
                    
                    print(string.format("\n✓ SUCCESS on attempt %d!", attempt))
                end
            end
        else
            print(string.format("Attempt %d: Request failed (%.3f seconds)", attempt, request_time or 0))
        end
        
        -- Small delay between requests (1 millisecond) if not successful
        if not success then
            socket.sleep(0.001)  -- 1 millisecond delay using luasocket
        end
    end
    
    -- Get final end time using luaposix gettimeofday
    local end_tv = posix.gettimeofday()
    local end_time = end_tv.sec + end_tv.usec / 1000000.0
    local total_time = end_time - start_time
    
    -- Print results
    print("\n" .. string.rep("=", 60))
    print("RESULTS")
    print(string.rep("=", 60))
    print(string.format("Time to first successful request: %.3f seconds", total_time))
    print(string.format("First success timestamp: %s", os.date("!%Y-%m-%dT%H:%M:%SZ", first_success_timestamp)))
    print(string.format("Total attempts: %d", attempt))
    print(string.format("Final status code: %d", final_status))
    if final_body then
        local body_preview = final_body:sub(1, 200)
        if #final_body > 200 then
            body_preview = body_preview .. "..."
        end
        print(string.format("Response body preview: %s", body_preview))
    end
    print(string.rep("=", 60))
    
    -- Save results to file
    local output_dir = os.getenv("OUTPUT_DIR")
    if output_dir then
        -- Create directory recursively if it doesn't exist using luaposix
        local ok, err = mkdir_p(output_dir)
        if not ok then
            print(string.format("Warning: Could not create directory %s: %s", output_dir, err or "unknown error"))
        end
        
        -- Prepare result data
        local result_data = {
            time_to_first_success = total_time,
            first_success_timestamp = first_success_timestamp,
            first_success_timestamp_iso = os.date("!%Y-%m-%dT%H:%M:%SZ", first_success_timestamp),
            total_attempts = attempt,
            final_status_code = final_status,
            endpoint = {
                method = method,
                path = path,
                full_url = full_url
            },
            response_body = final_body
        }
        
        -- Save as JSON
        local result_file = output_dir .. "/first_request_result.json"
        local file = io.open(result_file, "w")
        if file then
            file:write(json.encode(result_data))
            file:close()
            print(string.format("\nResults saved to: %s", result_file))
        else
            print(string.format("\nWarning: Could not write results to %s", result_file))
        end
    else
        print("\nNote: OUTPUT_DIR not set, results not saved to file")
    end
end

-- Run main function
local ok, err = pcall(main)
if not ok then
    print(string.format("Error: %s", err))
    os.exit(1)
end



