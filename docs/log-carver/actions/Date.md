# Date Action

The `date` action uses patterns to parse a timestamp from a field and store it as the event's [`@timestamp`](../../Events.md#-timestamp).

- [Date Action](#date-action)
  - [Example](#example)
  - [Options](#options)
    - [`field`](#field)
    - [`formats`](#formats)
    - [`remove`](#remove)

## Example

```yaml
- name: date
  field: timestamp
  remove: true
  formats:
    - '02-01-2006 15:04:05'
    - 'Jan 02, 2006 3:04:05 PM'
```

## Options

### `field`

String. Required

The name of the field to parse. Use `[]` to access nested fields, for example `nested[field]`.

### `formats`

Array of Strings. Required

A list of time formats that will be attempted in order. As soon as the field is successfully parsed the action completes and does not attempt any further formats.

A time format is specified by writing the reference time `Mon Jan 2 15:04:05 MST 2006` in the format you desire, such as `2006-01-02`, or `2nd January`. This works as all numerical components are distinct from each other (See: [Golang "time" package constants](https://golang.org/pkg/time/#pkg-constants))

### `remove`

Boolean. Optional. Default: false

If set to true, the parsed field will be unset from the event after parsing completes.
