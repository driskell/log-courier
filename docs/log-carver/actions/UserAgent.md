# User Agent Action

The `useragent` action parses a HTTP User-Agent header into browser, OS and device information.

A new `user_agent` field will be added to the event after a successful parse and will contain the following nested fields. The `user_agent` field attempts to match the Elastic Common Schema (ECS).

- `original`. String
- `name`. String

The following nested fields are also present if data is available, and are otherwise omitted.

- `device[name]`. String.
- `major`. String
- `minor`. String
- `patch`. String
- `os[family]`. String
- `os[major]`. String
- `os[minor]`. String
- `os[version]`. String

- [User Agent Action](#user-agent-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)
    - [`remove`](#remove)

## Example

```yaml
- name: useragent
  field: useragent
```

## Options

### `field`

String. Required

The name of the field to parse. Use `[]` to access nested fields, for example `nested[field]`.

### `remove`

Boolean. Optional. Default: false

If set to true, the parsed field will be unset from the event after parsing completes.
