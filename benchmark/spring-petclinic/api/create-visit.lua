-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local newVisitRequest = 'date=2021-02-23&description=Broken+leg'
local _, method, headers, _ = OpeanApiParse.getRequestParameters(os.getenv("SAMPLE_NAME"))

local threads = {}
local threadCount = 1
function setup(thread)
    thread:set("threadId", threadCount)
    threadCount = threadCount + 1
    table.insert(threads, thread)
end

local ownersAndPets = {}
local maxIndex
function parseOwnersAndPets()
    for line in io.lines(os.getenv("OUTPUT_DIR") .. string.format("/petclinic-petids-%d.txt", threadId)) do
        for str in string.gmatch(line, "([^;]+)") do
          table.insert(ownersAndPets, str)
        end
    end
    maxIndex = #ownersAndPets
end

local petId
local firstPetId
local lastPetId
local index
function init(args)
    parseOwnersAndPets()
    index = 1
end

function incrementIndex()
    index = index + 2
    if index > maxIndex then
        index = 1
    end
end

function request()
    local ownerId = ownersAndPets[index]
    local petId = ownersAndPets[index + 1]

    local path = string.format("/owners/%s/pets/%s/visits/new", ownerId, petId)
    local req = wrk.format(method, path, headers, newVisitRequest)

    incrementIndex()
    return req
end