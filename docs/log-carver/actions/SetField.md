# Set Field Action

The `set_field` action sets a field on the event to the specified value.

Note that some fields are [`reserved`](../../Events.md#reserved-fields) and cannot be set directly or can only accept certain values.

- [Set Field Action](#set-field-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)
    - [`value`](#value)

## Example

```yaml
- name: set_field
  field: fieldname
  value: string
```

## Options

### `field`

String. Required

The name of the field to set. Use `[]` to access nested fields, for example `nested[field]`.

### `value`

Expression. Required

The value to set using an [Expression](../Configuration.md#expression).
