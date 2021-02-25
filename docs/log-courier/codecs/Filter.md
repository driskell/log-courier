# Filter Codec

The `filter` codec strips out unwanted events, shipping only those desired.

- [Filter Codec](#filter-codec)
  - [Example](#example)
  - [Options](#options)
    - [`patterns`](#patterns)
    - [`match`](#match)

## Example

```yaml
- name: filter
  patterns:
  - "^(.*connect from.*)$"
  - "^(.*status=sent.*)$"
```

## Options

### `patterns`

Array of Strings. Required

A set of regular expressions to match against each line.

These are applied in the order that they are specified. As soon as the required
number of matches occurred (dictated by the `match` configuration that defaults
to `any`), the event is shipped. Patterns with higher hit rates should be
specified first when `match` is `any`.

The pattern syntax is detailed at <https://code.google.com/p/re2/wiki/Syntax.>

To negate a pattern such that a line is considered to match when the pattern
does not match, prefix the pattern with an exclamation mark ("!"). For example,
the pattern "!^uninteresting" would match any line which does not start with the
text, "uninteresting".

The pattern can also be prefixed with "=" to explicitly state that a pattern is
not negated, which then allows a literal match of an exclamation mark at the
start of the pattern. For example, "=!useful!" would match a line containing,
"!useful!".

### `match`

String. Optional. Default: "any"  
Available values: "any", "all"

Specifies whether matching a single pattern will ship an event, or if all
patterns must match before shipping occurs.
