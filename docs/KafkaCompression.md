# Kafka Compression

When the `Kafka` deployment model is used, the eBPF agent produces flow records to a Kafka topic and flowlogs-pipeline consumes them. Compression can be configured via `spec.kafka.compression` to reduce network bandwidth and Kafka storage at the cost of some CPU on the producer side.

## Configuration

```yaml
apiVersion: flows.netobserv.io/v1beta2
kind: FlowCollector
spec:
  deploymentModel: Kafka
  kafka:
    address: "kafka-cluster-kafka-bootstrap.netobserv:9093"
    topic: "network-flows"
    compression: "lz4"   # default
```

The `compression` field accepts: `none`, `gzip`, `snappy`, `lz4`, `zstd`.

The same field is available on Kafka exporters (`spec.exporters[].kafka.compression`).

## How it works

- **Producer side (eBPF agent):** messages are compressed per batch before being sent to Kafka. The compression codec is stored in the message headers.
- **Consumer side (flowlogs-pipeline):** decompression is handled automatically by the Kafka client based on message headers. No configuration is needed on the consumer.
- **Kafka brokers:** compressed messages are stored as-is on disk, reducing storage. Brokers do not need any specific configuration for client-side compression.

## Codec comparison

| Codec | Compression ratio | CPU cost (producer) | Decompression cost | Notes |
|-------|:-----------------:|:-------------------:|:------------------:|-------|
| `none` | 1x | — | — | No overhead; use when CPU is the bottleneck |
| `lz4` | ~2–3x | Very low | Very low | **Recommended default.** Best latency/ratio trade-off |
| `snappy` | ~2–3x | Very low | Very low | Similar to lz4, slightly lower ratio |
| `zstd` | ~3–5x | Moderate | Low | Higher ratio; good for high-throughput clusters |
| `gzip` | ~3–5x | High | Moderate | Maximum ratio but significant CPU cost |

Flow records (protobuf-encoded) contain many repeated fields (IP addresses, namespaces, node names) and compress well, often reaching the higher end of these ranges.

## Recommendations

- **`lz4` (default):** best choice for most deployments. The CPU overhead is negligible (~microseconds per batch) even on busy nodes, while providing meaningful bandwidth reduction. Since the eBPF agent runs as a DaemonSet on every node, keeping per-node CPU impact low is important.

- **`zstd`:** consider this for large clusters with high flow rates where the Kafka network link or storage is the bottleneck. The additional CPU cost is moderate and concentrated on the eBPF agent pods.

- **`none`:** use this only if the eBPF agent pods are severely CPU-constrained and bandwidth is not a concern (e.g., same-node Kafka or very low flow rates).

- **`gzip`/`snappy`:** generally not recommended over `lz4`/`zstd`, but supported for compatibility with environments that require a specific codec.

## Impact on batching

Compression efficiency improves with larger batches. The `spec.agent.ebpf.kafkaBatchSize` setting (default 1 MiB) controls the maximum batch size in bytes before compression. Larger batches compress better but add latency. The default batch size provides a good balance.

## Sources

The compression ratios and CPU cost estimates in the codec comparison table are approximate values derived from upstream benchmarks, not from NetObserv-specific measurements. Actual results will vary depending on flow record characteristics and batch sizes.

- [lz4 benchmarks](https://github.com/lz4/lz4) — Yann Collet's reference benchmarks
- [Zstandard benchmarks](https://github.com/facebook/zstd) — Meta's comparison across compression algorithms
- [KIP-110: Add Codec for ZStandard Compression](https://cwiki.apache.org/confluence/display/KAFKA/KIP-110%3A+Add+Codec+for+ZStandard+Compression) — Kafka proposal comparing codec trade-offs
- [Snappy documentation](https://github.com/google/snappy) — Google's design goals and performance characteristics
