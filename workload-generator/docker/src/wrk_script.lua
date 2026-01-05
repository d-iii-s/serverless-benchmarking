local json = require("json")
local data_generator = require("data-generator")

-- Thread tracking for aggregation in done()
local threads = {}
local thread_counter = 0

function setup(thread)
    -- Store thread reference for later aggregation
    table.insert(threads, thread)
end

function init(args)
    sessions = {}
    
    -- Thread-local status tracking (use different names to avoid conflicts)
    my_status_counts = {}
    my_total_requests = 0
    my_total_errors = 0
    
    wrk.thread:set("id", thread_counter)
    thread_counter = thread_counter + 1

    data_generator.load_scenario("scenario.json")
    vertex_count = data_generator.get_vertex_count()
    wrk.thread:set("vertex_count", vertex_count)
end

function request(conn_id)
    local session = get_session(sessions, conn_id)
    local path, method, body, content_type, new_unique_values_generated, generated_values = data_generator.build_request(
        nil,
        session.responses,
        session.generated_values,
        session.step,
        wrk.thread:get("id"),
        conn_id,
        session.unique_values_generated
    )
    session.unique_values_generated = new_unique_values_generated
    session.generated_values[session.step] = generated_values

    return wrk.format(method, path, {
        ["Content-Type"] = content_type
    }, body)
end

function response(status, headers, body, conn_id)
    local session = get_session(sessions, conn_id)
    local vertex_count = wrk.thread:get("vertex_count")
    session.responses[session.step] = body
    
    -- Track response status codes for histogram (thread-local)
    my_total_requests = my_total_requests + 1
    local status_key = tostring(status)
    if not my_status_counts[status_key] then
        my_status_counts[status_key] = 0
    end
    my_status_counts[status_key] = my_status_counts[status_key] + 1
    
    -- Track errors (4xx and 5xx)
    if status >= 400 then
        my_total_errors = my_total_errors + 1
    end
    
    -- Store in thread-local storage for aggregation
    wrk.thread:set("thread_status_counts", json.encode(my_status_counts))
    wrk.thread:set("thread_total_requests", my_total_requests)
    wrk.thread:set("thread_total_errors", my_total_errors)
    
    if session.step + 1 == vertex_count then
        sessions[conn_id] = new_session()
    else
        session.step = session.step + 1
    end
end

-------------------------------------------------------
-- Done callback - aggregates and outputs response status histogram
-------------------------------------------------------

function done(summary, latency, requests)
    -- Aggregate data from all threads
    local aggregated_status_counts = {}
    local aggregated_total_requests = 0
    local aggregated_total_errors = 0
    
    for _, thread in ipairs(threads) do
        local thread_requests = thread:get("thread_total_requests") or 0
        local thread_errors = thread:get("thread_total_errors") or 0
        local thread_status_json = thread:get("thread_status_counts") or "{}"
        
        aggregated_total_requests = aggregated_total_requests + thread_requests
        aggregated_total_errors = aggregated_total_errors + thread_errors
        
        -- Parse and merge status counts
        local thread_status_counts = json.decode(thread_status_json)
        if thread_status_counts then
            for status, count in pairs(thread_status_counts) do
                if not aggregated_status_counts[status] then
                    aggregated_status_counts[status] = 0
                end
                aggregated_status_counts[status] = aggregated_status_counts[status] + count
            end
        end
    end
    
    -- Sort status codes for consistent output
    local sorted_statuses = {}
    for status, _ in pairs(aggregated_status_counts) do
        table.insert(sorted_statuses, status)
    end
    table.sort(sorted_statuses, function(a, b) return tonumber(a) < tonumber(b) end)
    
    -- Print summary to stdout
    io.write("\n--- RESPONSE STATUS HISTOGRAM ---\n")
    io.write(string.format("Total requests: %d\n", aggregated_total_requests))
    io.write(string.format("Total errors (4xx/5xx): %d\n", aggregated_total_errors))
    io.write(string.format("Threads aggregated: %d\n", #threads))
    
    if aggregated_total_requests > 0 then
        io.write("\nStatus code distribution:\n")
        for _, status in ipairs(sorted_statuses) do
            local count = aggregated_status_counts[status]
            local percentage = (count / aggregated_total_requests) * 100
            local bar_length = math.floor(percentage / 2)
            local bar = string.rep("█", bar_length)
            io.write(string.format("  %s: %8d (%6.2f%%) %s\n", status, count, percentage, bar))
        end
    end
    
    -- Build histogram data
    local histogram_data = {
        total_requests = aggregated_total_requests,
        total_errors = aggregated_total_errors,
        threads = #threads,
        status_codes = aggregated_status_counts
    }
    
    -- Write JSON to file in OUTPUT_DIR
    local output_dir = os.getenv("OUTPUT_DIR") or "."
    local output_path = output_dir .. "/response_histogram.json"
    local file = io.open(output_path, "w")
    if file then
        file:write(json.encode(histogram_data))
        file:close()
        io.write(string.format("\nHistogram data written to: %s\n", output_path))
    else
        io.write(string.format("\nWarning: Could not write histogram file to %s\n", output_path))
    end
end

-------------------------------------------------------
-- Helper Functions
-------------------------------------------------------

function new_session()
    return {
        step = 0,
        responses = {},
        generated_values = {},
        unique_values_generated = 0
    }
end

function get_session(sessions, conn_id)
    if not sessions[conn_id] then
        sessions[conn_id] = new_session()
    end
    return sessions[conn_id]
end

function dump(o)
    if type(o) == 'table' then
        local s = '{ '
        for k,v in pairs(o) do
            if type(k) ~= 'number' then k = '"'..k..'"' end
            s = s .. '['..k..'] = ' .. dump(v) .. ','
        end
        return s .. '} '
    else
        return tostring(o)
    end
end
