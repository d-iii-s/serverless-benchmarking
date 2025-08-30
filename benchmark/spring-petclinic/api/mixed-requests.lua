-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local newOwnerPath, newOwnerMethod, newOwnerHeaders, _ = OpeanApiParse.getRequestParameters("exampleNewOwner")
local _, newPetMethod, newPetHeaders, _ = OpeanApiParse.getRequestParameters("exampleNewPet")
local getOwnerPath, getOwnerMethod, getOwnerHeaders, getOwnerBody = OpeanApiParse.getRequestParameters("ownersExample")
local _, newVisitMethod, newVisitHeaders, newVisitBody = OpeanApiParse.getRequestParameters("exampleNewVisit")

local petTypes = {"bird", "cat", "dog", "hamster", "lizard", "snake"}
local newOwnerRequest = 'firstName=F%d&lastName=L%d&address=A%d&city=C%d&telephone=%d'
local newPetRequest = 'name=Pet%d&birthDate=2010-10-05&type=%s'

local threadIdCounter = 1
function setup(thread)
    thread:set("threadId", threadIdCounter)
    threadIdCounter = threadIdCounter + 1
end

local ownerCounter
local petCounter
function init(args)
   ownerCounter = 1000000000000000 + 10000000000000 * threadId
   petCounter = ownerCounter
end

local ownerPath
local petId
local state = 0
-- default data
local ownerIdToPetId = {
    [1] = {1},
    [2] = {2},
    [3] = {3,4},
    [4] = {5},
    [5] = {6},
    [6] = {7,8},
    [7] = {9},
    [8] = {10},
    [9] = {11},
    [10] = {12, 13}
}
function request()
    if state == 0 then
        -- create a new owner at http://localhost:8006/owners/new
        ownerCounter = ownerCounter + 1
        method = newOwnerMethod
        path = newOwnerPath
        headers = newOwnerHeaders
        body = string.format(newOwnerRequest, ownerCounter, ownerCounter, ownerCounter, ownerCounter, ownerCounter / 10000000)
    elseif state == 1 then
        -- create a new pet for the current owner at http://localhost:8006/owners/%d/pets/new
        petCounter = petCounter + 1
        method = newPetMethod
        ownerPath = string.format("/owners/%d", 1)
        path = ownerPath .. "/pets/new"
        headers = newPetHeaders
        body = string.format(newPetRequest, petCounter, petTypes[(petCounter % #petTypes) + 1])
    elseif state == 2 then
        -- navigate to http://localhost:8006/owners/%d
        method = getOwnerMethod
        path = getOwnerPath
        headers = getOwnerHeaders
        body = getOwnerBody
    elseif state == 3 then
        -- create a new visit for the current pet at http://localhost:8006/owners/%d/pets/%d/visits/new
        method = newVisitMethod
        ownerId = 1
        ownerPath = string.format("/owners/%d", ownerId)
        petId = ownerIdToPetId[ownerId][1]
        path = ownerPath .. "/pets/" .. petId .. "/visits/new"
        headers = newVisitHeaders
        body = newVisitBody
    elseif state == 4 then
        -- navigate to http://localhost:8006/owners/%d
        method = getOwnerMethod
        path = getOwnerPath
        headers = getOwnerHeaders
        body = getOwnerBody
    end
    
    state = (state + 1) % 5
    return wrk.format(method, path, headers, body)
end