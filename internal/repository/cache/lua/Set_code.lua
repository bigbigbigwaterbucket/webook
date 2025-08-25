---@diagnostic disable: undefined-global
--你的验证码在redis上的key
--phone_code:login:1314xxx
--从redis的lua脚本调用函数那会传来keys变量和argv变量
--！！！lua脚本是从1开始计数的
local key = KEYS[1]
--phone_code:login:1314xxx:cnt
local cntKey = key..":cnt"
local val = ARGV[1]

local ttl= tonumber(redis.call("ttl",key))  --获取redis中key的过期时间，顺带来判断key是否存在

if ttl ==-1 then
    -- key存在，但是没有过期时间
    return -2
elseif ttl==-2 or ttl<540 then
    -- key不存在或者没有在一分钟内重发过
    redis.call("set",key,val)
    redis.call("expire",key,600)
    redis.call("set",cntKey,3) --失败次数最多三次
    redis.call("expire",cntKey,600)
    return 0
else
    --key过期时间没经过1min
    return -1

end
