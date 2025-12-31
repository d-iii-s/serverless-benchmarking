local json = require("json")
local data_generator = require("data-generator")
local thread_counter = 0

function init(args)
    sessions = {}
    
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
    --print(dump(sessions))
    if session.step + 1 == vertex_count then
        sessions[conn_id] = new_session()
    else
        session.step = session.step + 1
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