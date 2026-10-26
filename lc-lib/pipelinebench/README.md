# Pipeline Benchmarks

Benchmarks for the log-carver pipeline, run in-process with the same wiring as
`log-carver/log-carver.go`: receiver pool → spooler → processor → publisher.

Only the edges are simulated:

- **Connections** feed the receiver pool directly. Their payloads decode JSON on
  every `Events()` call, as the courier protocol messages do.
- **The output transport** sits behind the real publisher. Each of its routines
  encodes a payload, waits a configured latency, then acknowledges it.

## Benchmarks

| Benchmark | One op is | Reports |
| --- | --- | --- |
| `BenchmarkPipelineThroughput` | One event through the whole pipeline to an output that acknowledges instantly, with `b.N` events split across 1 or 100 connections | `ns/op` per delivered event, `allocs/op` per event, `events/s`, `closes` |
| `BenchmarkPipelineFairness` | One 5 second scenario (1 second with `-short`) of 5 heavy senders (1024 event payloads, 1 or 10 in flight) and 295 light senders (10 event payloads, 1 in flight) against an output with 10ms latency | Throughput and acknowledgement latency per sender class |

Similar throughput results across connection counts indicate a serial stage
limits the pipeline.

The fairness scenario suffixed `/slow=1in50x250ms` makes one in every 50
payloads take 250ms at the output, as a retried bulk request would. While
acknowledgements are released in a single global order, a slow payload holds
back the acknowledgements of every payload behind it. Around 2% of light
payloads sit inside a slow payload and wait for it whatever the ordering, which
sets `light-p99-ms`, so the payloads held back behind it show in `light-p90-ms`
and `light-p95-ms`.

Fairness metrics:

| Metric | Meaning |
| --- | --- |
| `events/s`, `heavy-events/s`, `light-events/s` | Events acknowledged per second, overall and per sender class |
| `heavy-p50-ms`, `light-p50-ms`, `light-p90-ms`, `light-p95-ms`, `light-p99-ms`, `light-p999-ms` | Time from sending a payload to its full acknowledgement. `light-p999-ms` shows a minority of light connections being starved, and is used rather than the maximum, which a single pause can set |
| `light-wait-ratio` | `light-p50-ms` divided by `heavy-p50-ms`. Near 1 when light senders wait as long as heavy ones despite sending far less, and lower when they are served ahead of heavy backlogs. It isolates fairness from speed, as faster processing lowers both latencies but leaves the ratio unchanged |
| `closes` | Connections closed by the receiver, for example when the queue is full. A closed connection stops sending, so also reported by throughput, where it means fewer events were delivered |

Only payloads sent before the scenario deadline are recorded. Load continues
until every recorded payload is acknowledged so the tail of the run sees the
same conditions, then connections drain and disconnect.

## Running

Throughput lets Go choose `b.N`. Fairness is one long scenario per op, so runs
once per count:

```sh
go test ./lc-lib/pipelinebench -run '^$' -bench Throughput -benchmem -count 5
go test ./lc-lib/pipelinebench -run '^$' -bench Fairness -benchtime 1x -count 5
```

Add `-cpuprofile`, `-memprofile`, `-blockprofile` or `-mutexprofile` to see
where time, allocations or waiting occur.

## Reference report

`run.sh` runs these and the related micro benchmarks into a single report,
`latest.txt`, headed with the commit, date and Go version, and compares it with
the committed `reference.txt` using a pinned
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat):

```sh
lc-lib/pipelinebench/run.sh                     # run, then compare with the reference
lc-lib/pipelinebench/run.sh --diff              # compare the last run without running again
lc-lib/pipelinebench/run.sh --update-reference  # make the last run the new reference
```

The comparison shows the full benchstat table, where `~` marks a change that is
not statistically significant, followed by a summary of only the significant
changes. Changes of at least `THRESHOLD` percent (default 10) are listed first,
each marked better or worse for its metric. `COUNT` sets the runs per benchmark
(default 10), as benchstat needs at least 4 to detect a change and 6 for a
confidence interval.

Results depend on the machine, so only compare reports from the same hardware;
the report records the OS, architecture and CPU. Commit `reference.txt` with a
change to make it the new reference, keeping past references in its history.

## Related micro benchmarks

| Package | Benchmarks | One op is |
| --- | --- | --- |
| `lc-lib/event` | `BenchmarkEventBytes`, `BenchmarkEventDataFromBytes` | Encoding or decoding one event |
| `lc-lib/spooler` | `BenchmarkSpoolerEncodedEvents`, `BenchmarkSpoolerUnencodedEvents` | Spooling one batch of 128 events, with or without an existing encoding |
| `lc-lib/transports/tcp` | `BenchmarkConnectionReadCompressed` | Decompressing one 1024 event payload read through a connection over a real loopback socket |
| `lc-lib/transports/tcp/courier` | `BenchmarkProtocolSendEvents*` | Handing one 1024 event payload to a connection, with or without an existing encoding |
| `lc-lib/transports/tcp/courier` | `BenchmarkProtocolReadEVNT` | Reading one 1024 event EVNT message from a connection, including decompression |
