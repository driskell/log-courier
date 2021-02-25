# Unset Field Action

The `unset_field` action sets a field on the event to the specified value.

Note that some fields are [`reserved`](../../Events.md#reserved-fields) and cannot be unset.

- [Unset Field Action](#unset-field-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)

## Example

```yaml
- name: unset_field
  field: fieldname
```

## Options

### `field`

String. Required

The name of the field to unset. Use `[]` to access nested fields, for example `nested[field]`.
