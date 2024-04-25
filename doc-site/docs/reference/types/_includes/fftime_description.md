Times are serialized to JSON on the API in RFC 3339 / ISO 8601 nanosecond UTC time
for example `2022-05-05T21:19:27.454767543Z`.

Note that JavaScript can parse this format happily into millisecond time with `Date.parse()`.

Times are persisted as a nanosecond resolution timestamps in the database.

On input, and in queries, times can be parsed from RFC3339, or unix timestamps
(second, millisecond or nanosecond resolution).