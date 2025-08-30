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
local firstOwnerId
local lastOwnerId

function init(args)
    parseOwnerIdLimits()

    local numOwnerIds = lastGlobalOwnerId - firstGlobalOwnerId + 1
    local ownerIdsPerThread = math.floor(numOwnerIds / threadCount)
    firstOwnerId = firstGlobalOwnerId + ownerIdsPerThread * (threadId - 1)
    lastOwnerId = firstOwnerId + ownerIdsPerThread - 1

    ownerId = firstOwnerId
end

function incrementOwnerId()
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