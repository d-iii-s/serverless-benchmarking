-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")
local pathPdf, methodPdf, headersPdf, bodyPdf = OpeanApiParse.getRequestParameters("samplePDF")
local pathOdt, methodOdt, headersOdt, bodyOdt = OpeanApiParse.getRequestParameters("sampleODT")

local state = 0
function request()
    local path, method, body, headers
    
    if state == 0 then
        -- odt request
        path = pathOdt
        method = methodOdt
        headers = headersOdt
        body = bodyOdt
    elseif state == 1 then
        -- pdf request
        path = pathPdf
        method = methodPdf
        headers = headersPdf
        body = bodyPdf
    end

    state = (state + 1) % 2
    return wrk.format(method, path, headers, body)
end
