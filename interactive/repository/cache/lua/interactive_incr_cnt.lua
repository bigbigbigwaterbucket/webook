---@diagnostic disable: undefined-global
local key = KEYS[1]
-- map数据结构里边的key，标识是阅读/收藏还是啥
local cntKey = ARGV[1]
local delta = tonumber(ARGV[2])
-- 顶多一个key
local exists = redis.call("EXISTS", key)
if exists == 1 then
    redis.call("HINCRBY", key, cntKey, delta)
    -- 说明自增成功了
    return 1
else
    return 0
end
