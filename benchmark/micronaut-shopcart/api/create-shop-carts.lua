-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local path, method, headers, _ = OpeanApiParse.getRequestParameters("newClient")

local newUserRequest = "{ \"username\": \"%d\", \"name\": \"%dname\" }"

local threads = {}
local threadIdCounter = 1
function setup(thread)
   thread:set("threadId", threadIdCounter)
   threadIdCounter = threadIdCounter + 1
   table.insert(threads, thread)
end

function getFirstUserId(threadId)
    return 1000000000000000 + 10000000000000 * threadId
end

function init(args)
    userId = getFirstUserId(threadId)
end

function request()
    body = string.format(newUserRequest, userId, userId)
    userId = userId + 1
    return wrk.format(method, path, headers, body)
end

function done(summary, latency, requests)
    file = io.open(os.getenv("OUTPUT_DIR") .. "/shopcart-userids.txt", "w")
    for index, thread in ipairs(threads) do
        local threadId  = thread:get("threadId")
        local firstUserId = getFirstUserId(threadId)
        local lastUserId = thread:get("userId")
        file:write(string.format("%d;%d\n", firstUserId, lastUserId))
    end
    file:close()
end
