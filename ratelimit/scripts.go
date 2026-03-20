package ratelimit

// Lua script for atomic multi-key rate limit check and increment.
// Checks all limits first, then increments only if all pass.
// KEYS: list of counter keys to check
// ARGV: alternating (limit, increment, ttl) triples for each key
// Returns: [allowed(0/1), deny_index(-1 if allowed), remaining_1, remaining_2, ...]
const luaCheckAndIncrement = `
local n = #KEYS
local results = {}
local denied = false
local deny_index = -1

-- Phase 1: Read all current values and check limits
for i = 1, n do
    local current = tonumber(redis.call('GET', KEYS[i]) or '0')
    local limit = tonumber(ARGV[(i-1)*3 + 1])
    local increment = tonumber(ARGV[(i-1)*3 + 2])

    if current + increment > limit then
        denied = true
        deny_index = i - 1
        break
    end
    results[i] = limit - current - increment
end

-- Phase 2: If all passed, increment all counters
if not denied then
    for i = 1, n do
        local increment = tonumber(ARGV[(i-1)*3 + 2])
        local ttl = tonumber(ARGV[(i-1)*3 + 3])
        local new_val = redis.call('INCRBY', KEYS[i], increment)
        redis.call('EXPIRE', KEYS[i], ttl)
        results[i] = tonumber(ARGV[(i-1)*3 + 1]) - new_val
    end
end

-- Build return array
local ret = {}
ret[1] = denied and 0 or 1
ret[2] = deny_index
for i = 1, n do
    ret[i + 2] = results[i] or 0
end
return ret
`

// Lua script for acquiring a concurrency slot.
// KEYS[1]: concurrency counter key
// ARGV[1]: max concurrency limit
// ARGV[2]: TTL in seconds (safety net)
// Returns: 1 if acquired, 0 if denied
const luaConcurrencyAcquire = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local limit = tonumber(ARGV[1])
if current >= limit then
    return 0
end
redis.call('INCR', KEYS[1])
redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
return 1
`

// Lua script for releasing a concurrency slot.
// KEYS[1]: concurrency counter key
// Returns: new count
const luaConcurrencyRelease = `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
if current > 0 then
    return redis.call('DECR', KEYS[1])
end
return 0
`
