-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local addBananaRequest = "{ \"username\": \"%d\", \"name\": \"bananas\", \"amount\": \"7\" }"
local addJoghurtRequest = "{ \"username\": \"%d\", \"name\": \"joghurt\", \"amount\": \"12\" }"
local addJoghurtRequest = "{ \"username\": \"%d\", \"name\": \"joghurt\", \"amount\": \"12\" }"
local addCoffeeRequest = "{ \"username\": \"%d\", \"name\": \"coffee\", \"amount\": \"5\" }"
local addCheeseRequest = "{ \"username\": \"%d\", \"name\": \"cheese\", \"amount\": \"5\" }"
local addMeatRequest = "{ \"username\": \"%d\", \"name\": \"meat\", \"amount\": \"800\" }"

local path, method, headers, _ = OpeanApiParse.getRequestParameters("addBananas")

function parseUserIdLimits()
    local result = {}
    for line in io.lines(os.getenv("OUTPUT_DIR") .. "/shopcart-userids.txt") do
        local userIdLimit = {}
        for str in string.gmatch(line, "([^;]+)") do
          table.insert(userIdLimit, tonumber(str))
        end
        table.insert(result, userIdLimit)
    end
    return result
end

local userIdLimits = parseUserIdLimits()
local threadIdCounter = 1
function setup(thread)
    local userIdLimit = userIdLimits[threadIdCounter]
    thread:set("firstUserId", userIdLimit[1])
    thread:set("lastUserId", userIdLimit[2])
    threadIdCounter = threadIdCounter + 1
end

local userId
function init(args)
    userId = firstUserId
end

function incrementUserId()
    userId = userId + 1
    if userId > lastUserId then
        userId = firstUserId
    end
end

local state = 0
function request()
    if state == 0 then
        body = string.format(addBananaRequest, userId)
    elseif state == 1 then
        body = string.format(addJoghurtRequest, userId)
    elseif state == 2 then
        body = string.format(addCoffeeRequest, userId)
    elseif state == 3 then
        body = string.format(addCheeseRequest, userId)
    elseif state == 4 then
        body = string.format(addMeatRequest, userId)
        incrementUserId()
    end
   
    state = (state + 1) % 5
    return wrk.format(method, path, headers, body)
end
