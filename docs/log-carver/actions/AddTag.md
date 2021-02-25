# Add Tag Action

The `add_tag` action adds a tag to the event's reserved [`tags`](../../Events.md#tags) field, which always exists and is always an array of strings.

- [Add Tag Action](#add-tag-action)
  - [Example](#example)
  - [Options](#options)
    - [`tag`](#tag)

## Example

```yaml
- name: add_tag
  tag: tagname
```

## Options

### `tag`

String. Required

The tag to add to the list. If the tag already exists the action still succeeds but does nothing.
