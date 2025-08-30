local posix = require("posix")

local thread_counter = 1
local header_pattern = "%d,%d"
local threads = {}

function setup(thread)
  thread:set("id", thread_counter)
  table.insert(threads, thread)
  thread_counter = thread_counter + 1
end

function init(args)
    requests = ""
    responses = ""
    thread_request_counter = 0
end

function request()
    local thread_id = wrk.thread:get("id");

    -- Get the current time
    local timespec = posix.time.clock_gettime(0)
    -- Extract seconds and nanoseconds
    local seconds = timespec.tv_sec
    local nanoseconds = timespec.tv_nsec
    -- Convert nanoseconds to milliseconds
    local milliseconds = math.floor(nanoseconds / 1e6)
    -- Convert seconds to hours, minutes, and seconds
    local time = os.date("*t", seconds)
    local hours = time.hour
    local minutes = time.min
    local seconds_in_minute = time.sec
    -- Print time in hours:minutes:seconds.milliseconds format
    local send_time = string.format("%02d:%02d:%02d.%03d", hours, minutes, seconds_in_minute, milliseconds)

--     local send_time = os.date("%Y-%m-%d %H:%M:%S")
    wrk.headers["X-counter"] = string.format(header_pattern, thread_id, thread_request_counter)
    -- io.write(string.format("%s,%d,%d\n", send_time, thread_id, thread_request_counter))
    -- table.insert(requests, string.format("%s,%d,%d\n", send_time, thread_id, thread_request_counter))

    -- csv format date, thread_number, request_number
    requests = requests .. string.format("%s,%d,%d\n", send_time, thread_id, thread_request_counter)
    thread_request_counter = thread_request_counter + 1
    return wrk.request()
end

function response(status, headers, body)
    local thread_id = wrk.thread:get("id")

    -- Get the current time
    local timespec = posix.time.clock_gettime(0)
    -- Extract seconds and nanoseconds
    local seconds = timespec.tv_sec
    local nanoseconds = timespec.tv_nsec
    -- Convert nanoseconds to milliseconds
    local milliseconds = math.floor(nanoseconds / 1e6)
    -- Convert seconds to hours, minutes, and seconds
    local time = os.date("*t", seconds)
    local hours = time.hour
    local minutes = time.min
    local seconds_in_minute = time.sec
    -- Print time in hours:minutes:seconds.milliseconds format
    local receive_time = string.format("%02d:%02d:%02d.%03d", hours, minutes, seconds_in_minute, milliseconds)

--     local receive_time = os.date("%Y-%m-%d %H:%M:%S")
    thread_number_request_number = wrk.headers["X-counter"]
    -- csv format date, thread_number, request_number
    -- io.write(string.format("%s,%s\n", receive_time, thread_number_request_number))
    -- table.insert(responses, string.format("%s,%s\n", receive_time, thread_number_request_number))

    responses = responses .. string.format("%s,%s\n", receive_time, thread_number_request_number)
end

-- todo: saving to file from os.env
function done(summary, latency, requests)
    local result_dir = os.getenv("RESULT_DIR")
    for thread_idx, thread in ipairs(threads) do
        local file_name = string.format("%s/thread-%d-requests.log", result_dir, thread_idx)
        local file, err = io.open(file_name, "w")
        if err then
            io.write(err)
            return
        end

        file:write("Request Log:\n")
        -- for _, entry in pairs(thread:get("requests")) do
        --     file:write(entry)
        -- end
        file:write(thread:get("requests"))
        file:close()

        local file_name = string.format("%s/thread-%d-responses.log", result_dir, thread_idx)
        local file, err = io.open(file_name, "w")
        if err then
            io.write(err)
            return
        end
        file:write("Response Log:\n")
        -- for _, entry in pairs(thread:get("responses")) do
        --     file:write(entry)
        -- end
        file:write(thread:get("responses"))
        file:close()
    end
    -- for thread_idx, thread in ipairs(threads) do
    --     local file_name = string.format("temp/thread-%d-requests.log", thread_idx)
    --     local file, err = io.open(file_name, "w")
    --     if err then
    --         io.write(err)
    --         return
    --     end

    --     file:write("Request Log:\n")
    --     for _, entry in pairs(thread:get("requests")) do
    --         io.write(entry)
    --         file:write(entry)
    --     end
    --     file:close()

    --     local file_name = string.format("temp/thread-%d-responses.log", thread_idx)
    --     local file, err = io.open(file_name, "w")
    --     if err then
    --         io.write(err)
    --         return
    --     end
    --     file:write("Response Log:\n")
    --     for _, entry in pairs(thread:get("responses")) do
    --         io.write(entry)
    --         file:write(entry)
    --     end
    --     file:close()
    -- end
end