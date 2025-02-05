Large integers of up to 256bits in size are common in blockchain, and
handled in FireFly.

In JSON output payloads in FireFly, including events, they are serialized as
strings (with base 10).

On input you can provide JSON string (string with an `0x` prefix are
parsed at base 16), or a JSON number.

## Maximum size of numbers in versions of FireFly up to `v1.3.1`

In versions of FireFly up to and including `v1.3.1`, be careful when using large JSON numbers. The largest number that is safe to transfer using a JSON number is 2^53 - 1 and it is
possible to receive errors from the transaction manager, or for precision to be silently lost when passing numeric parameters larger than that. It is recommended to pass large numbers as strings to avoid loss of precision.

## Maximum size of numbers in versions of FireFly `v1.3.2` and higher

In FireFly `v1.3.2` support was added for 256-bit precision JSON numbers. Some application frameworks automatically serialize large JSON numbers to a string which FireFly already supports, but there is no upper limit
to the size of a number that can be represented in JSON. FireFly now supports much larger JSON numbers, up to 256-bit precision. For example the following input parameter to a contract constructor is now supported:

```
    ...
    "definition": [{
		"inputs": [
			{
				"internalType":" uint256",
				"name": "x",
				"type": "uint256"
			}
		],
		"outputs":[],
		"type":"constructor"
	}],
	"params": [ 10000000000000000000000000 ]
    ...
```

Some application frameworks seralize large numbers in scientific notation e.g. `1e+25`. FireFly `v1.3.2` added supported for handling scientific numbers in parameters. This removes the need to change an application
that uses this number format. For example the following input parameter to a contract constructor is now supported:

```
    ...
    "definition": [{
		"inputs": [
			{
				"internalType":" uint256",
				"name": "x",
				"type": "uint256"
			}
		],
		"outputs":[],
		"type":"constructor"
	}],
	"params": [ 1e+25 ]
    ...
```
