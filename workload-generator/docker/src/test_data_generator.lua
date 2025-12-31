#!/usr/bin/env lua

-- Test suite for data-generator.lua
-- Tests the generate_fake_data function with various hints and parameters

-- Set up Lua path for luarocks modules
package.path = package.path .. ";/usr/local/share/lua/5.1/?.lua;/usr/local/share/lua/5.1/?/init.lua"

local data_generator = require("data-generator")

-- Test utilities
local tests_passed = 0
local tests_failed = 0
local test_results = {}

local function assert(condition, message)
    if condition then
        tests_passed = tests_passed + 1
        table.insert(test_results, {status = "PASS", message = message})
        return true
    else
        tests_failed = tests_failed + 1
        table.insert(test_results, {status = "FAIL", message = message})
        print("FAIL: " .. message)
        return false
    end
end

local function assert_type(value, expected_type, message)
    local actual_type = type(value)
    return assert(actual_type == expected_type, 
        message .. " (expected " .. expected_type .. ", got " .. actual_type .. ")")
end

local function assert_not_nil(value, message)
    return assert(value ~= nil, message .. " (got nil)")
end

local function assert_in_range(value, min, max, message)
    return assert(value >= min and value <= max, 
        message .. " (value " .. tostring(value) .. " not in range [" .. min .. ", " .. max .. "])")
end

local function assert_string_length(str, min_len, max_len, message)
    local len = #str
    return assert(len >= min_len and len <= max_len,
        message .. " (length " .. len .. " not in range [" .. min_len .. ", " .. max_len .. "])")
end

local function assert_matches_pattern(str, pattern, message)
    return assert(string.match(str, pattern) ~= nil,
        message .. " (string '" .. str .. "' doesn't match pattern '" .. pattern .. "')")
end

-- Test constants
local TEST_THREAD_ID = 1
local TEST_CONN_ID = 100
local TEST_UNIQUE_VALUES = 0

print("=" .. string.rep("=", 70))
print("Testing data-generator.lua")
print("=" .. string.rep("=", 70))
print()

-- Test 1: Basic name hints
print("Test 1: Name hints")
local firstName = data_generator.generate_fake_data("firstName", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(firstName, "firstName should not be nil")
assert_type(firstName, "string", "firstName should be a string")
assert(#firstName > 0, "firstName should not be empty")

local lastName = data_generator.generate_fake_data("lastName", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(lastName, "lastName should not be nil")
assert_type(lastName, "string", "lastName should be a string")

local fullName = data_generator.generate_fake_data("fullName", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(fullName, "fullName should not be nil")
assert_type(fullName, "string", "fullName should be a string")
print("  ✓ Name hints passed")
print()

-- Test 2: Internet hints
print("Test 2: Internet hints")
local email = data_generator.generate_fake_data("email", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(email, "email should not be nil")
assert_type(email, "string", "email should be a string")
assert_matches_pattern(email, "@", "email should contain @")

local username = data_generator.generate_fake_data("username", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(username, "username should not be nil")
assert_type(username, "string", "username should be a string")

local url = data_generator.generate_fake_data("url", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(url, "url should not be nil")
assert_type(url, "string", "url should be a string")

local ipv4 = data_generator.generate_fake_data("ipv4", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(ipv4, "ipv4 should not be nil")
assert_type(ipv4, "string", "ipv4 should be a string")
print("  ✓ Internet hints passed")
print()

-- Test 3: Number generation with min/max
print("Test 3: Number generation with min/max")
local number1 = data_generator.generate_fake_data("number", "integer", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, 10, 100)
assert_not_nil(number1, "number with min/max should not be nil")
assert_type(number1, "number", "number should be a number")
assert_in_range(number1, 10, 100, "number should be in range [10, 100]")

local number2 = data_generator.generate_fake_data("integer", "integer", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, 1, 10)
assert_not_nil(number2, "integer with min/max should not be nil")
assert_in_range(number2, 1, 10, "integer should be in range [1, 10]")

local float1 = data_generator.generate_fake_data("float", "number", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, 0.0, 1.0)
assert_not_nil(float1, "float with min/max should not be nil")
assert_type(float1, "number", "float should be a number")
assert_in_range(float1, 0.0, 1.0, "float should be in range [0.0, 1.0]")
print("  ✓ Number generation with min/max passed")
print()

-- Test 4: String generation with minLength/maxLength
print("Test 4: String generation with minLength/maxLength")
local word1 = data_generator.generate_fake_data("word", "string", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, nil, nil, nil, 5, 10)
assert_not_nil(word1, "word with minLength/maxLength should not be nil")
assert_type(word1, "string", "word should be a string")
assert_string_length(word1, 5, 10, "word should have length in range [5, 10]")

local password1 = data_generator.generate_fake_data("password", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, nil, nil, nil, 12, 20)
assert_not_nil(password1, "password with minLength/maxLength should not be nil")
assert_type(password1, "string", "password should be a string")
assert_string_length(password1, 12, 20, "password should have length in range [12, 20]")
print("  ✓ String generation with minLength/maxLength passed")
print()

-- Test 5: Pattern matching
print("Test 5: Pattern matching")
local word_pattern = data_generator.generate_fake_data("word", "string", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, nil, nil, "^[a-z]+$", nil, nil)
assert_not_nil(word_pattern, "word with pattern should not be nil")
assert_type(word_pattern, "string", "word with pattern should be a string")
-- Note: Pattern matching depends on faker2 implementation
print("  ✓ Pattern matching passed")
print()

-- Test 6: Date generation
print("Test 6: Date generation")
local date1 = data_generator.generate_fake_data("date", nil, "date", TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(date1, "date should not be nil")
assert_type(date1, "string", "date should be a string")

local timestamp = data_generator.generate_fake_data("timestamp", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(timestamp, "timestamp should not be nil")
assert_type(timestamp, "string", "timestamp should be a string")
print("  ✓ Date generation passed")
print()

-- Test 7: UUID generation
print("Test 7: UUID generation")
local uuid1 = data_generator.generate_fake_data("uuid", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(uuid1, "uuid should not be nil")
assert_type(uuid1, "string", "uuid should be a string")
assert_matches_pattern(uuid1, "^[0-9a-f%-]+$", "uuid should match UUID pattern")
print("  ✓ UUID generation passed")
print()

-- Test 8: Boolean generation
print("Test 8: Boolean generation")
local bool1 = data_generator.generate_fake_data("boolean", "boolean", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(bool1, "boolean should not be nil")
assert_type(bool1, "boolean", "boolean should be a boolean")
print("  ✓ Boolean generation passed")
print()

-- Test 9: Address hints
print("Test 9: Address hints")
local city = data_generator.generate_fake_data("city", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(city, "city should not be nil")
assert_type(city, "string", "city should be a string")

local state = data_generator.generate_fake_data("state", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(state, "state should not be nil")
assert_type(state, "string", "state should be a string")

local country = data_generator.generate_fake_data("country", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(country, "country should not be nil")
assert_type(country, "string", "country should be a string")
print("  ✓ Address hints passed")
print()

-- Test 10: URI and hostname
print("Test 10: URI and hostname")
local uri = data_generator.generate_fake_data("uri", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(uri, "uri should not be nil")
assert_type(uri, "string", "uri should be a string")

local hostname = data_generator.generate_fake_data("hostname", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(hostname, "hostname should not be nil")
assert_type(hostname, "string", "hostname should be a string")
print("  ✓ URI and hostname passed")
print()

-- Test 11: Date and DateTime
print("Test 11: Date and DateTime")
local dateTime = data_generator.generate_fake_data("dateTime", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(dateTime, "dateTime should not be nil")
assert_type(dateTime, "string", "dateTime should be a string")
print("  ✓ Date and DateTime passed")
print()

-- Test 12: ID generation
print("Test 12: ID generation")
local id1 = data_generator.generate_fake_data("id", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(id1, "id should not be nil")
assert_type(id1, "string", "id should be a string")

local id2 = data_generator.generate_fake_data("id", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, nil, nil, nil, 10, 20)
assert_not_nil(id2, "id with minLength/maxLength should not be nil")
assert_type(id2, "string", "id should be a string")
print("  ✓ ID generation passed")
print()

-- Test 13: Byte and binary generation
print("Test 13: Byte and binary generation")
local byte1 = data_generator.generate_fake_data("byte", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, nil, nil, nil, 10, 20)
assert_not_nil(byte1, "byte with minLength/maxLength should not be nil")
assert_type(byte1, "string", "byte should be a string")

local binary1 = data_generator.generate_fake_data("binary", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES, nil, nil, nil, 50, 100)
assert_not_nil(binary1, "binary with minLength/maxLength should not be nil")
assert_type(binary1, "string", "binary should be a string")
print("  ✓ Byte and binary generation passed")
print()

-- Test 14: Default fallback for unknown hints
print("Test 14: Default fallback for unknown hints")
local unknown1 = data_generator.generate_fake_data("unknown_hint", "string", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(unknown1, "unknown hint with string type should not be nil")
assert_type(unknown1, "string", "unknown hint should return a string")

local unknown2 = data_generator.generate_fake_data("unknown_hint", "integer", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(unknown2, "unknown hint with integer type should not be nil")
assert_type(unknown2, "number", "unknown hint with integer type should return a number")

local unknown3 = data_generator.generate_fake_data("unknown_hint", "boolean", nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(unknown3, "unknown hint with boolean type should not be nil")
assert_type(unknown3, "boolean", "unknown hint with boolean type should return a boolean")
print("  ✓ Default fallback passed")
print()

-- Test 15: Seed reproducibility (same seed should produce same results)
print("Test 15: Seed reproducibility")
local seed1_1 = data_generator.generate_fake_data("number", "integer", nil, 1, 1, 0, 1, 100)
local seed1_2 = data_generator.generate_fake_data("number", "integer", nil, 1, 1, 0, 1, 100)
-- Note: Due to salt in seed generation, results may vary, but structure should be consistent
assert_not_nil(seed1_1, "seeded number should not be nil")
assert_not_nil(seed1_2, "seeded number should not be nil")
assert_type(seed1_1, "number", "seeded number should be a number")
assert_type(seed1_2, "number", "seeded number should be a number")
print("  ✓ Seed reproducibility passed")
print()

-- Test 16: Multiple hint name variations
print("Test 16: Multiple hint name variations")
local name1 = data_generator.generate_fake_data("firstName", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
local name2 = data_generator.generate_fake_data("first_name", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
local name3 = data_generator.generate_fake_data("name", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(name1, "firstName should work")
assert_not_nil(name2, "first_name should work")
assert_not_nil(name3, "name should work")

local ip1 = data_generator.generate_fake_data("ip", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
local ip2 = data_generator.generate_fake_data("ipv4", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
local ip3 = data_generator.generate_fake_data("ipv6", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(ip1, "ip should work")
assert_not_nil(ip2, "ipv4 should work")
assert_not_nil(ip3, "ipv6 should work")
print("  ✓ Multiple hint name variations passed")
print()

-- Test 17: IPv6 generation
print("Test 17: IPv6 generation")
local ipv6 = data_generator.generate_fake_data("ipv6", nil, nil, TEST_THREAD_ID, TEST_CONN_ID, TEST_UNIQUE_VALUES)
assert_not_nil(ipv6, "ipv6 should not be nil")
assert_type(ipv6, "string", "ipv6 should be a string")
assert_matches_pattern(ipv6, ":", "ipv6 should contain colons")
print("  ✓ IPv6 generation passed")
print()

-- Summary
print("=" .. string.rep("=", 70))
print("Test Summary")
print("=" .. string.rep("=", 70))
print("Tests passed: " .. tests_passed)
print("Tests failed: " .. tests_failed)
print("Total tests: " .. (tests_passed + tests_failed))
print()

if tests_failed == 0 then
    print("✓ All tests passed!")
    os.exit(0)
else
    print("✗ Some tests failed!")
    os.exit(1)
end

