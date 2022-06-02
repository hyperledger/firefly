Large integers of up to 256bits in size are common in blockchain, and
handled in FireFly.

In JSON output payloads in FireFly, including events, they are serialized as
strings (with base 10).

On input you can provide JSON string (string with an `0x` prefix are
parsed at base 16), or a JSON number.

Be careful when using JSON numbers, that the largest
number that is safe to transfer using a JSON number is 2^53 - 1.