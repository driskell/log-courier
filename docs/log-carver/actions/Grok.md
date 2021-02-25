# Grok Action

The `grok` action parses a field using a set of patterns, extracting matches into new fields in the event.

It uses the [RE2](https://code.google.com/p/re2/wiki/Syntax) regular expression library provided by Golang, which completes in linear time, and for the purposes of Grok is extremely fast and significantly more performant than other variants, with only a minor reduction in functionality. Of particular note is the `%{GREEDYDATA}` match (see the [`patterns`](#patterns) syntax) can be used freely with little to no impact on performance, whereas other grok libraries would suffer exponential growth in compute-time.

When converting patterns previously used in Logstash, the main difference is that look-behind assertions and look-forward assertions are not supported by RE2, and so minor rewrites may be necessary to use some patterns with Log Carver.

- [Grok Action](#grok-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)
    - [`local patterns`](#local-patterns)
    - [`patterns`](#patterns)
    - [`remove`](#remove)

## Example

```yaml
- name: grok
  field: message
  remove: true
  patterns:
  - '(?P<timestamp>%{MONTH} +%{MONTHDAY} %{TIME}) (?:<%{NONNEGINT:facility}.%{NONNEGINT:priority}> )?%{IPORHOST:logsource}+(?: %{PROG:program}(?:\[%{POSINT:pid}\])?:|) %{GREEDYDATA:message}'
```

## Options

### `field`

String. Required

The name of the field to parse. Use `[]` to access nested fields, for example `nested[field]`.

### `local patterns`

Provides an optional additional set of named patterns that the [`patterns`](#patterns) configuration can reference. This can be used to simplify and annotate a large complex pattern by splitting it into smaller named components.

Local patterns use the same syntax as [`patterns`](#patterns).

If a [builtin pattern](../../../lc-lib/grok/builtin.go) already exists with the same name, the `local patterns` version will take precedence.

For example:

```yaml
- name: grok
  field: message
  local patterns:
    TIMESTAMP: '(?P<timestamp>%{MONTH} +%{MONTHDAY} %{TIME})'
  patterns:
  - '%{TIMESTAMP} (?:<%{NONNEGINT:facility}.%{NONNEGINT:priority}> )?%{IPORHOST:logsource}+(?: %{PROG:program}(?:\[%{POSINT:pid}\])?:|) %{GREEDYDATA:message}'
```

### `patterns`

Array of Strings. Required

A list of patterns that will be attempted one by one. The first one to match will stop any further matching, and fields will be generated and added to the event from the named matches within the matching pattern.

The pattern syntax is the [RE2 Syntax](https://code.google.com/p/re2/wiki/Syntax).

[Builtin patterns](../../../lc-lib/grok/builtin.go) (or local patterns) can be referenced using the syntax `%{NAME}`. To store a builtin pattern's match into a field, use `%{NAME:field}`, which will store the match into a field named `field`. Use `[]` syntax to reference a nested field, such as `%{NAME:nested[field]}`. This is similiar to how grok is configured with Logstash, and many of Logstash's patterns exist in Log Carver as builtin patterns. Most grok patterns can be made up entirely of these builtin patterns, such as `%{NUMBER}`, `%{QUOTEDSTRING}` and `%{GREEDYDATA}`.

To ensure the correct type of data is stored in the target field, it can be coerced by specifying the desired type in the reference using the syntax `%{NAME:field:type}` where `type` is one of `int`, `string` or `float`.

All named matches (`(?P<name>match)`) from the pattern will also be stored into new fields in the event with the same name, and will always result in a string. To coerce a named match you will need to set it up in [`local patterns`](#local-patterns) so you can reference it using the `%{NAME:field:type}` syntax.

### `remove`

Boolean. Optional. Default: false

If set to true, the parsed field will be unset from the event after parsing completes.
