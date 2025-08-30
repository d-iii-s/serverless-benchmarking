-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local threads = {}
local threadCount = 1
local _, method, headers, body = OpeanApiParse.getRequestParameters(os.getenv("SAMPLE_NAME"))

function setup(thread)
    thread:set("threadId", threadCount)
    threadCount = threadCount + 1
    table.insert(threads, thread)
end

local firstGlobalOwnerId = math.huge
local lastGlobalOwnerId = -1
function parseOwnerIdLimits()
    for line in io.lines(os.getenv("OUTPUT_DIR") .. "/petclinic-ownerids.txt") do
        local ownerIds = {}
        for str in string.gmatch(line, "([^;]+)") do
          table.insert(ownerIds, tonumber(str))
        end
        
        firstGlobalOwnerId = math.min(firstGlobalOwnerId, ownerIds[1])
        lastGlobalOwnerId = math.max(lastGlobalOwnerId, ownerIds[2])
    end
end

local ownerId
local prevOwnerId
local firstOwnerId
local lastOwnerId
local file
function init(args)
    parseOwnerIdLimits()

    local numOwnerIds = lastGlobalOwnerId - firstGlobalOwnerId + 1
    local ownerIdsPerThread = math.floor(numOwnerIds / threadCount)
    firstOwnerId = firstGlobalOwnerId + ownerIdsPerThread * (threadId - 1)
    lastOwnerId = firstOwnerId + ownerIdsPerThread - 1

    ownerId = firstOwnerId
    file = io.open(os.getenv("OUTPUT_DIR") .. string.format("/petclinic-petids-%d.txt", threadId), "w")
end

function incrementOwnerId()
    prevOwnerId = ownerId
    ownerId = ownerId + 1
    if ownerId > lastOwnerId then
        ownerId = firstOwnerId
    end
end

function request()
    local path = string.format("/owners/%d", ownerId)
    incrementOwnerId()
    return wrk.format(method, path, headers, body)
end

function response(status, headers, body)
    for str in string.gmatch(body, 'pets/(%d+)/visits/new') do
        -- It would be nice to use a thread-local table instead but with wrk2 it is not possible to access a thread-local
        -- table in the function 'done' (that only works with wrk). Writing the file directly is significantly faster than
        -- using string concatenation.
        file:write(string.format("%d;%s\n", prevOwnerId, str))
    end
end