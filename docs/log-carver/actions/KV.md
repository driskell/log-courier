# KV Action

The `kv` action parses a key-value string into fields. It parses a space-separated list of key=value pairs. Keys can also refer to nested fields. Values can be quoted using single or double quotes or can be unquoted if they contain no space.

For example, the field value `number=500 field="value in quotes" nested[entry]=value` would add the following new fields to the event.

```yaml
number: 500
field: value in quotes
nested:
  entry: value
```

- [KV Action](#kv-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)
    - [`prefix`](#prefix)

## Example

```yaml
- name: kv
  field: data
```

## Options

### `field`

String. Required

The name of the field to parse. Use `[]` to access nested fields, for example `nested[field]`.

### `prefix`

Pattern String. Optional

Prefixes all added fields with the given prefix. The prefix can contain values from fields in the event using the [`Pattern String`](../Configuration.md#pattern-string) syntax.

With a value of `prefix_`, the field value `number=100 nested[field]="testing"` would add the following new fields to the event.

```yaml
prefix_number: 100
prefix_nested:
  field: testing
```
