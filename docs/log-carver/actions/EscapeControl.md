# Escape Control Action

The `escape_control` action escapes terminal control characters in event fields so they are safe to display. Control characters (other than tab and newline, and including DEL and the C1 range) are escaped as `\xHH`, for example ESC as `\x1B`, and any backslash in the same value is escaped as `\\` so the value can be reversed. Values without control characters are left unchanged. Field names within nested fields are escaped too. If anything is escaped, the event is tagged with `_control_chars`, and with `_control_chars_<field>` for each field that had any: each selected field, or each top-level field when escaping the entire event.

Every event is escaped after the pipeline has run while [`escape control characters`](../Configuration.md#escape-control-characters) is enabled, so this action is useful to escape fields earlier in the pipeline, such as before conditionals that test them, or to escape only some fields when that option is disabled.

- [Escape Control Action](#escape-control-action)
  - [Example](#example)
  - [Options](#options)
    - [`fields`](#fields)

## Example

```yaml
- name: escape_control
  fields:
    - message
```

## Options

### `fields`

Array of Strings. Optional. Default: all fields

The fields to escape, including all values nested within them. Use `[]` to access nested fields, for example `nested[field]`, and a backslash to use `[`, `]` or `\` literally in a name, for example `nested\[field\]`. If not specified, the entire event is escaped.
