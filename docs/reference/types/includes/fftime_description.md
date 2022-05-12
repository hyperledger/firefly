FFTime is serialized to JSON on the API in RFC3339 nanosecond UTC time
(noting that JavaScript can parse this format happily into millisecond time with Date.pase()).
It is persisted as a nanosecond resolution timestamp in the database.
It can be parsed from RFC3339, or unix timestamps (second, millisecond or nanosecond resolution).