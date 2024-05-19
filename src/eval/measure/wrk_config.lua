wrk.method = "POST"
wrk.headers["content-type"] = "application/json"
wrk.body = "{ \"response_size\": 800 }"

-- local file = io.open("latencies.log", "w")

-- Function to format the current timestamp
-- local function get_timestamp()
--   return os.date("%Y-%m-%d %H:%M:%S")
-- end

-- -- Override the response function to log the latency
-- response = function(status, headers, body)
--   local timestamp = get_timestamp()
--   file:write(string.format("%s\n", timestamp))
-- end

-- Close the file when the script finishes
-- function done(summary, latency, requests)
--   file:write("---")
--   for i = 1, #latency do
--       file:write(string.format("%d\n", latency[i]))
--   end
--   file:close()
-- end

-- function done(summary, latency, requests)
--     -- Iterate over each latency value and print it with timestamps
--     local file = io.open("latencies.log", "w")
--     file:write("DONE\n")
--     for i = 1, #latency do
--         print(string.format("Request %d: Latency %d ms", i, latency[i]))
--     end
--     file:close()
-- end

local ts_file = io.open("timestamps.txt", "w")
local lat_file = io.open("latencies.txt", "w")
-- Function to format the current timestamp
local function get_timestamp()
    return tostring(os.microtime())
end

-- Override the response function to log the latency
response = function(status, headers, body)
    local timestamp = get_timestamp()
    ts_file:write(string.format("%s\n", timestamp))
end

-- Override the done function to write to the file
function done(summary, latency, requests)
    -- Iterate over each latency value and write it to the file
    for i = 1, #latency do
        lat_file:write(string.format("Request %d: Latency %d ms\n", i, latency[i]))
    end
    -- Close the file handle after writing
    lat_file:close()
    ts_file:close()
end