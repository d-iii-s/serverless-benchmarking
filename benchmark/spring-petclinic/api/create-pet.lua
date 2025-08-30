-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local petTypes = {"bird", "cat", "dog", "hamster", "lizard", "snake"}
local newPetRequest = "name=Pet%d&birthDate=2010-10-05&type=%s"
local _, method, headers, _ = OpeanApiParse.getRequestParameters(os.getenv("SAMPLE_NAME"))

local firstGlobalOwnerId = math.huge
local lastGlobalOwnerId = -1
function parseOwnerIdLimits()
    for line in io.lines(os.getenv("OUTPUT_DIR") .. "/petclinic-ownerids.txt") do
        local ownerIds = {}
        for str in string.gmatch(line, "([^;]+)") do
          table.insert(ownerIds, str)
        end
        
        firstGlobalOwnerId = math.min(firstGlobalOwnerId, ownerIds[1])
        lastGlobalOwnerId = math.max(lastGlobalOwnerId, ownerIds[2])
    end
end

local threads = {}
local threadCount = 1
function setup(thread)
    thread:set("threadId", threadCount)
    threadCount = threadCount + 1
    table.insert(threads, thread)
end


local ownerId
local firstOwnerId
local lastOwnerId
local petCounter

function init(args)
    parseOwnerIdLimits()

    local numOwnerIds = lastGlobalOwnerId - firstGlobalOwnerId + 1
    local ownerIdsPerThread = math.floor(numOwnerIds / threadCount)
    firstOwnerId = firstGlobalOwnerId + ownerIdsPerThread * (threadId - 1)
    lastOwnerId = firstOwnerId + ownerIdsPerThread - 1

    ownerId = firstOwnerId
    petCounter = 1000000000000000 + 10000000000000 * threadId
end

function incrementOwnerId()
    ownerId = ownerId + 1
    if ownerId > lastOwnerId then
        ownerId = firstOwnerId
    end
end

function request()
    local path = string.format("/owners/%d/pets/new", ownerId)
    local body = string.format(newPetRequest, petCounter, petTypes[(petCounter % #petTypes) + 1])
    petCounter = petCounter + 1
    incrementOwnerId()
    return wrk.format(method, path, headers, body)
end