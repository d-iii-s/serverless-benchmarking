-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

local viewCartPath = "/cart/%d"

local _, method, headers, body = OpeanApiParse.getRequestParameters("getCartExample")

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

function request()
    path = string.format(viewCartPath, userId)
    incrementUserId()
    return wrk.format(method, path, headers, body)
end

