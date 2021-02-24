# Multiline Codec

The multiline codec processes multiple lines into a single event.

As long as lines match the specified `pattern` they are buffered. When a line is
encountered that does not match, an event is flushed as dictated by the `what`
option.

- [Multiline Codec](#multiline-codec)
  - [Example](#example)
  - [Options](#options)
    - [`"max multiline bytes"`](#%22max-multiline-bytes%22)
    - [`"patterns"`](#%22patterns%22)
    - [`"match"`](#%22match%22)
    - [`"previous timeout"`](#%22previous-timeout%22)
    - [`"what"`](#%22what%22)

## Example

```json
{
  "name": "multiline",
  "patterns": ["!^[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} "],
  "what": "previous",
  "previous timeout": "30s"
}
```

## Options

### `"max multiline bytes"`

Number. Optional. Default: `spool max bytes`

The maximum multiline length to process. If a multiline block exeeds this
length, it will be split across multiple events.

This setting can not be greater than the `spool max bytes` setting.

### `"patterns"`

Array of Strings. Required

A list of regular expressions to match against each line.

These are applied in the order that they are specified. As soon as the required
number of matches occurred (dictated by the `match` configuration that defaults
to `any`), the pattern is considered matched, and the action specified by `what`
takes place.

The pattern syntax is detailed at <https://code.google.com/p/re2/wiki/Syntax.>

To negate a pattern such that a line is considered to match when the pattern
does not match, prefix the pattern with an exclamation mark ("!"). For example,
the pattern "!^STARTEVENT" would match any line which does not start with the
text, "STARTEVENT".

The pattern can also be prefixed with "=" to explicitly state that a pattern is
not negated, which then allows a literal match of an exclamation mark at the
start of the pattern. For example, "=!EVENT!" would match a line containing,
"!EVENT!".

### `"match"`

String. Optional. Default: "any"  
Available values: "any", "all"

Specifies whether matching a single pattern must be matched or if all patterns
must be matched.

### `"previous timeout"`

Duration. Optional. Default: 0. Ignored when "what" != "previous"

When using `"previous"`, if `"previous timeout"` is not 0 any buffered lines
will be flushed as a single event if no more lines are received within the
specified time period.

### `"what"`

*String. Optional. Default: "previous"  
Available values: "previous", "next"*

- `"previous"`: When the line matches, it belongs in the same event as the
previous line. In other words, when matching stops treat the current line as the
start of the next event. Flush the previously buffered lines as a single event
and start a new buffer containing this line.
- `"next"`: When the line matches, it belongs in the same event as the next
line. In other words, when matching stops treat the current line as the end of
the current event. Flush the previously buffered lines along with this line as a
single event and start a new buffer.

A side effect of using `"previous"` is that an event will not be flushed until
the first line of the next event is encountered. The `"previous timeout"` option
offers a solution to this.
