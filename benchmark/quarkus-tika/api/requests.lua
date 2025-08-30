-- Author: Artem Bakhtin
local OpeanApiParse = require("parser")

function init(args)
   local path, method, headers, body = OpeanApiParse.getRequestParameters(os.getenv("SAMPLE_NAME"))
   wrk.path = path
   wrk.method = method
   wrk.headers = headers
   wrk.body = body
end