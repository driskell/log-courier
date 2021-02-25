# Events

Events withing Log Courier and Log Carver have a common structure.

- [Events](#events)
  - [Format](#format)
  - [Reserved Fields](#reserved-fields)
    - [@metadata](#metadata)
    - [@timestamp](#timestamp)
    - [tags](#tags)

## Format

Events generated are, by default, in the following format which is much like Filebeat and the original Logstash Forwarder.

```yaml
"@timestamp": 2021-01-01T01:02:03Z00:00
host: localhost.localdomain
message: This is the line data from the file and was bigger than max line bytes and therefore chopped in the mi
offset: 10223
path: /var/log/file.log
tags:
- splitline
timezone: +0000 UTC
```

Some of these fields are not enabled by default, or have varying formats, as controlled by the [Stream Configuration](log-courier/Configuration.md#stream-configuration) parameters for the file.

If the [`enable ecs`](log-courier/Configuration.md#enable-ecs) stream configuration parameter is change to true, the format will change to the below. Please read the `enable ecs` documentation carefully as **it is not backwards compatible**. Log Carver itself will expect this format too so you should enable this option when using Log Carver (see [`Index Template`](log-carver/Configuration.md#index-template) in the Log Carver documentation.)

```yaml
"@timestamp": 2021-01-01T01:02:03Z00:00
event:
  timezone: +0000 UTC
host:
  name: localhost.localdomain
  hostname: localhost.localdomain
log:
  file:
    path: /var/log/file.log
  offset: 10223
message: This is the line data from the file and was bigger than max line bytes and therefore chopped in the mi
tags:
- splitline
```

## Reserved Fields

There are a specific set of reserved fields within an event that exhibit special behaviour. These are documented below, will always exist for every event, and cannot be unset within a Log Carver pipeline.

### @metadata

All events have a reserved field called `@metadata`. This is never transmitted over the wire and can therefore be used to store temporary information about the event during processing with Log Carver. For example, it could be used to store the target index the document should be stored in, and referenced in the [`index pattern`](log-carver/Configuration.md#index-pattern) configuration.

If an event is created which contains this as a field, such as from a JSON file, or from a fields configuration, the value will be lost and discarded.

### @timestamp

This is the timestamp of the event. When an event is created, such as from a JSON file, this field must be in the RFC3339 format, otherwise it will be replaced with the current time. Log Courier will automatically generate this field with the read time of the event.

### tags

This is an array of strings attached to the event. It is most commonly used to record processing errors. For example, a [Grok](log-carver/actions/Grok.md) failure will add a `_grok_failure` tag. A maximum of 1024 tags can exist on an event - any attempt to go beyond this limit will be silently ignored.
