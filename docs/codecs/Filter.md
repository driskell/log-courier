# Filter Codec

The filter codec strips out unwanted events, shipping only those desired.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Example](#example)
- [Options](#options)
  - [`"negate"`](#negate)
  - [`"patterns"`](#patterns)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Example

	{
		"name": "filter",
		"patterns": [ "^(.*connect from.*)$", "^(.*status=sent.*)$" ]
	}

## Options

### `"negate"`

*Boolean. Optional. Default: false*

Negates `patterns` so that an event is only shipped if none of the patterns
matched.

### `"patterns"`

*Array of Strings. Required*

A set of regular expressions to match against each line.

These are applied in the order that they are specified. As soon as a matching
pattern is found the event is shipped and the remaining patterns are skipped
until the next event. As such, patterns with higher hit rates should be
specified first.

The pattern syntax is detailed at https://code.google.com/p/re2/wiki/Syntax.
