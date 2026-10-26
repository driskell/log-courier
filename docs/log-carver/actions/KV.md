# KV Action

The `kv` action parses a key-value string into fields. It parses a space-separated list of key=value pairs. Values can be quoted using single or double quotes or can be unquoted if they contain no separator. Repeated separators are skipped, and a key with no value is set to an empty string.

For example, the field value `number=500 field="value in quotes" empty=` would add the following new fields to the event.

```yaml
number: 500
field: value in quotes
empty: ""
```

- [KV Action](#kv-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)
    - [`nested`](#nested)
    - [`prefix`](#prefix)
    - [`separators`](#separators)
    - [`unquoted hex`](#unquoted-hex)

## Example

```yaml
- name: kv
  field: data
```

## Options

### `field`

String. Required

The name of the field to parse. Use `[]` to access nested fields, for example `nested[field]`, and a backslash to use `[`, `]` or `\` literally in a name, for example `nested\[field\]`.

### `nested`

Array of Strings. Optional

Keys whose values are themselves key-value strings. Each is parsed again using the same options, with fields prefixed by the key name and an underscore after the [`prefix`](#prefix), and the original value is not stored. For example, with `nested: [msg]` the value `pid=1 msg='pid=2 res=ok'` sets `pid`, `msg_pid` and `msg_res`.

### `prefix`

Pattern String. Optional

Prefixes all added fields with the given prefix. The prefix can contain values from fields in the event using the [`Pattern String`](../Configuration.md#pattern-string) syntax.

With a value of `prefix_`, the field value `number=100 field="testing"` would add the following new fields to the event.

```yaml
prefix_number: 100
prefix_field: testing
```

### `separators`

String. Optional. Default: `" "`

The characters that separate each key-value pair, any of which can be used. It must not be empty or contain `=`, quotes or backslash. For example, Linux audit logs separate enriched fields with a `\x1D` character, which can be parsed using `" \x1D"`.

### `unquoted hex`

Array of Strings. Optional

Keys whose unquoted values are decoded from uppercase hexadecimal, with NUL characters decoded as spaces. Quoted values, and values that are not valid hexadecimal or do not decode to valid UTF-8, are stored unchanged. This decodes the fields that Linux audit logs encode, such as `unquoted hex: [proctitle, cmd, comm, exe, cwd, name]`.
