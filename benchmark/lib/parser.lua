-- Author: Artem Bakhtin

local yaml = require("lyaml")

local OpeanApiParse = {}

local function read_file(path)
    local file = io.open(path, "r")
    if not file then
        error("Could not open file: " .. path)
    end
    local content = file:read("*all")
    file:close()
    return content
end

local function parse_openapi_spec(file_path)
    -- Read the YAML file
    local content = read_file(file_path)
    -- Parse YAML to Lua table
    local spec = yaml.load(content)
    return spec
end

-- base on sample name find approporiate path, method, contentType and example body (or example file)
function OpeanApiParse.getRequestParameters(sample_name)
    local spec = parse_openapi_spec(os.getenv("OUTPUT_DIR") .. "/api.yaml")
    for rawPath, pathTable in pairs(spec["paths"]) do
        for method, methodTable in pairs(pathTable) do
            if methodTable["requestBody"] == nil or methodTable["requestBody"]["content"] == nil then
                goto endOfMethodLoop
            end
            for content, contentTable in pairs(methodTable["requestBody"]["content"]) do
                for example, exampleTable in pairs(contentTable["examples"]) do
                    if example == sample_name then
                        local body
                        local path = rawPath
                        local headers = {}
                        if methodTable["parameters"] then
                            for _, v in pairs(methodTable["parameters"]) do
                                if v["in"] == "query" or v["in"] == "path" then
                                    path = string.gsub(path, "{" .. v["name"] .. "}", v["example"])
                                elseif v["in"] == "header" then
                                    headers[v["name"]] = v["example"]
                                end
                            end
                        end
                        if exampleTable["externalValue"] ~= nil then
                            local file = io.open(os.getenv("OUTPUT_DIR") .. "/" .. exampleTable["externalValue"], "rb")
                            body = file:read("*a")
                        else
                            body = exampleTable["value"]
                        end
                        headers["Content-Type"] = content
                        return path, string.upper(method), headers, body
                    end
                end
            end
            ::endOfMethodLoop::
        end
    end

    return nil, nil, nil, nil
end

-- return the module table
return OpeanApiParse


