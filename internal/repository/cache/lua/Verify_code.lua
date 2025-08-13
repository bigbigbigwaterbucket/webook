---@diagnostic disable: undefined-global
local key = KEYS[1]
local val = redis.call("get",key)
local inputCode = ARGV[1]
local cntKey = key..":cnt"
local cntVal = tonumber(redis.call("get",cntKey)) or 0  --防止过期了
if cntVal <=0 then
    -- 输错太多或者过期 0 或者用过了 -1
    return -1
elseif inputCode==val then
    --用完不能再用
    redis.call("set",cntKey,-1)
    return 0
else
    -- 验证码不对
    redis.call("decr",cntKey,-1)
    return -2
end
