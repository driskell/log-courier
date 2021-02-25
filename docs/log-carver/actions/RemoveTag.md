# Remove Tag Action

The `remove_tag` action removes a tag from the event's reserved [`tags`](../../Events.md#tags) field, which always exists and is always an array of strings.

- [Remove Tag Action](#remove-tag-action)
  - [Example](#example)
  - [Options](#options)
    - [`tag`](#tag)

## Example

```yaml
- name: remove_tag
  tag: tagname
```

## Options

### `tag`

String. Required

The tag to remove. If the tag does not currently exist the action still succeeds but does nothing.
