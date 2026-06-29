package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/interviewos/backend/internal/content"
)

// beTopicDef defines a Backend Engineering depth topic. Slug (prefixed "be-")
// is the natural key; concept_md carries our own multi-sentence summary.
type beTopicDef struct {
	Slug    string
	Name    string
	Summary string
	Concept string
	Diff    content.Difficulty
	Hours   float64
}

// backendEngineeringTopics is the seeded Backend Engineering pillar. It covers
// the depth areas a senior backend interview probes: storage engines and
// indexing, transactions and consistency, distributed-systems primitives,
// messaging, caching, networking, the runtime, and operational concerns. Each
// topic carries a real concept summary (not a placeholder) so the catalog and
// the generated study tasks are immediately useful.
var backendEngineeringTopics = []beTopicDef{
	{"kafka-event-streaming", "Kafka & Event Streaming", "Distributed, partitioned, replicated commit log.",
		"Kafka stores records in partitioned, append-only logs replicated across brokers; consumers track their own offsets so reads are cheap and replayable. Key topics: partitions and ordering guarantees, consumer groups and rebalancing, the ISR set and acks for durability, log compaction vs retention, and exactly-once semantics via idempotent producers and transactions.",
		content.DifficultyMedium, 4.0},
	{"redis-caching-patterns", "Redis & Caching Patterns", "In-memory data store and caching strategies.",
		"Redis is a single-threaded in-memory store with rich data types (strings, hashes, sorted sets, streams) used for caching, rate limiting, leaderboards, and locks. Cover cache-aside vs write-through vs write-behind, TTL and eviction policies (LRU/LFU/allkeys), the thundering-herd and cache-stampede problem, and persistence via RDB snapshots and the AOF log.",
		content.DifficultyMedium, 3.5},
	{"sql-query-optimization", "SQL & Query Optimization", "Reading query plans and tuning SQL.",
		"Query optimization starts with EXPLAIN ANALYZE: distinguish seq scans from index scans, understand nested-loop vs hash vs merge joins, and watch for row-estimate skew that misleads the planner. Cover selectivity and statistics, covering and partial indexes, avoiding N+1 queries, keyset (cursor) pagination over OFFSET, and when denormalization beats a join.",
		content.DifficultyHard, 4.0},
	{"nosql-data-models", "NoSQL Data Models", "Key-value, document, and wide-column stores.",
		"NoSQL trades relational generality for a model matched to an access pattern: key-value (DynamoDB, Redis) for O(1) lookups, document (MongoDB) for nested aggregates, and wide-column (Cassandra, HBase) for write-heavy, partition-keyed workloads. Cover denormalization and query-first modeling, single-table design, secondary indexes, and the consistency/availability trade-offs each engine exposes.",
		content.DifficultyMedium, 3.5},
	{"transactions-isolation", "Transactions & Isolation Levels", "ACID and the anomalies isolation prevents.",
		"A transaction is an atomic, consistent, isolated, durable unit of work. Isolation levels trade correctness for concurrency: read-uncommitted, read-committed, repeatable-read, and serializable each forbid more anomalies (dirty reads, non-repeatable reads, phantoms, write skew). Cover two-phase locking vs optimistic concurrency, serializable snapshot isolation, and how deadlocks arise and are resolved.",
		content.DifficultyHard, 3.5},
	{"mvcc", "MVCC", "Multi-version concurrency control.",
		"MVCC lets readers never block writers by keeping multiple row versions: each statement sees a consistent snapshot determined by transaction visibility rules. Cover Postgres tuple versioning with xmin/xmax, the need for VACUUM to reclaim dead tuples and prevent transaction-id wraparound, and how MVCC implements snapshot isolation and its anomalies.",
		content.DifficultyHard, 3.0},
	{"indexing-btree-lsm", "Indexing: B-Tree & LSM", "The two dominant on-disk index structures.",
		"B-trees keep sorted pages for read-optimized, in-place updates (Postgres, MySQL); LSM trees buffer writes in memory and flush sorted runs that compact in the background (Cassandra, RocksDB), trading read amplification for write throughput. Cover write vs read vs space amplification, bloom filters over SSTables, and how the choice follows the workload.",
		content.DifficultyHard, 3.5},
	{"replication", "Replication", "Copying data across nodes for HA and read scale.",
		"Replication provides durability, availability, and read scaling. Cover single-leader (failover, replication lag, read-your-writes), multi-leader (write conflicts and resolution), and leaderless/quorum (Dynamo-style R+W>N) topologies; synchronous vs asynchronous trade-offs; and how lag breaks monotonic-read and causal guarantees.",
		content.DifficultyHard, 3.5},
	{"partitioning-sharding", "Partitioning & Sharding", "Splitting data across nodes.",
		"Sharding scales storage and throughput beyond one machine. Cover range vs hash vs directory partitioning, hot-spot/hotkey mitigation, consistent hashing with virtual nodes for minimal reshuffling, rebalancing strategies, secondary-index partitioning (local vs global), and routing requests to the right shard.",
		content.DifficultyHard, 3.5},
	{"cap-pacelc", "CAP & PACELC", "Fundamental trade-offs in distributed systems.",
		"CAP says that under a network partition a system must sacrifice either consistency or availability. PACELC extends it: even without partitions (Else) a system trades latency against consistency. Cover why partition tolerance is non-negotiable in practice, where common datastores sit (CP vs AP), and how tunable consistency lets one system span the spectrum.",
		content.DifficultyMedium, 2.5},
	{"consensus-raft-paxos", "Consensus: Raft, Paxos, ZAB", "Agreeing on a value despite failures.",
		"Consensus lets a cluster agree on an ordered log despite crashes. Raft decomposes it into leader election, log replication, and safety, using terms and a committed-index majority; Paxos proves the same guarantees more abstractly; ZAB underpins ZooKeeper. Cover quorums, why an odd node count matters, linearizable reads, and typical uses: leader election, config, and distributed locks.",
		content.DifficultyHard, 4.0},
	{"load-balancing-nginx", "Load Balancing & Nginx", "Distributing traffic across backends.",
		"Load balancers spread requests for scale and availability. Cover L4 (transport) vs L7 (application) balancing, algorithms (round-robin, least-connections, consistent-hash, EWMA), health checks and connection draining, and sticky sessions. Nginx specifics: reverse proxy, upstream blocks, TLS termination, keepalive pools, and as an API gateway.",
		content.DifficultyMedium, 3.0},
	{"cdn", "CDN", "Edge content delivery and caching.",
		"A CDN serves content from edge PoPs near users to cut latency and offload origin. Cover origin pull vs push, cache keys and TTLs, Cache-Control and surrogate headers, purge/invalidation, anycast and geo-routing, and edge compute for dynamic content. Discuss cache-hit ratio as the core efficiency metric.",
		content.DifficultyEasy, 2.0},
	{"docker-containers", "Docker & Containers", "OS-level virtualization for packaging.",
		"Containers isolate processes using Linux namespaces (pid, net, mnt, user) and cgroups for resource limits, sharing the host kernel. Cover image layering and the build cache, multi-stage builds for slim images, the difference from VMs, networking and volumes, and the OCI image/runtime standards.",
		content.DifficultyMedium, 2.5},
	{"kubernetes", "Kubernetes", "Container orchestration at scale.",
		"Kubernetes reconciles declarative desired state via controllers. Cover the Pod/ReplicaSet/Deployment hierarchy, Services and Ingress for networking, the scheduler and resource requests/limits, liveness/readiness probes, ConfigMaps/Secrets, horizontal pod autoscaling, and rolling updates with health-gated rollout and rollback.",
		content.DifficultyHard, 4.0},
	{"networking-tcp-tls-http", "Networking: TCP/TLS/HTTP", "The web request stack, end to end.",
		"Cover the TCP three-way handshake, congestion and flow control, and head-of-line blocking; the TLS handshake, certificates, and how TLS 1.3 cuts a round trip; and HTTP evolution from 1.1 keep-alive to HTTP/2 multiplexing and HTTP/3 over QUIC. Tie it together with what happens when you type a URL and press enter.",
		content.DifficultyMedium, 3.5},
	{"linux-fundamentals", "Linux Fundamentals", "Processes, files, and the kernel boundary.",
		"Backend engineers debug on Linux. Cover the process lifecycle (fork/exec, signals, zombies), file descriptors and everything-is-a-file, the page cache and virtual memory, the system-call boundary, and core tooling: top/htop, ps, lsof, strace, and /proc for introspection.",
		content.DifficultyMedium, 3.0},
	{"concurrency-synchronization", "Concurrency & Synchronization", "Coordinating shared state safely.",
		"Concurrency bugs come from unsynchronized shared mutable state. Cover race conditions and data races, mutexes vs RW locks, condition variables and semaphores, atomics and memory ordering, deadlock (the four Coffman conditions) and livelock, and the higher-level alternatives: lock-free structures and message passing over shared memory.",
		content.DifficultyHard, 3.5},
	{"memory-performance-profiling", "Memory & Performance Profiling", "Finding and fixing the bottleneck.",
		"Performance work is measurement-first. Cover the memory hierarchy and cache locality, the USE method (utilization, saturation, errors), latency percentiles vs averages, CPU vs memory vs IO-bound diagnosis, flame graphs, and tools: perf, pprof, and eBPF. Emphasize profiling before optimizing and Amdahl's law.",
		content.DifficultyHard, 3.5},
	{"go-runtime", "Go Runtime: Scheduler, Goroutines, Channels", "How Go runs concurrent code.",
		"Go's runtime multiplexes goroutines onto OS threads with the G-M-P scheduler, parking and stealing work to keep cores busy and handling blocking syscalls via handoff. Cover growable goroutine stacks, channels and select for communication, the happens-before guarantees of the memory model, and how the netpoller integrates async IO.",
		content.DifficultyHard, 4.0},
	{"garbage-collection", "Garbage Collection", "Automatic memory reclamation and its cost.",
		"GC trades manual memory management for pause time and CPU. Cover tracing (mark-sweep, mark-compact) vs reference counting, generational and concurrent collectors, and the latency vs throughput trade-off. Go specifics: the concurrent tri-color mark-sweep collector, write barriers, GOGC and the pacer, and reducing allocation pressure with escape analysis and pooling.",
		content.DifficultyHard, 3.0},
	{"api-design-idempotency", "API Design & Idempotency", "Designing robust service interfaces.",
		"Good APIs are predictable and safe to retry. Cover REST resource modeling and correct status codes, idempotency keys for at-least-once delivery, pagination (cursor over offset), versioning strategies, rate limiting and backpressure, and the trade-offs of REST vs gRPC vs GraphQL for service-to-service vs client-facing APIs.",
		content.DifficultyMedium, 3.0},
	{"distributed-transactions-saga", "Distributed Transactions & Saga", "Consistency across services.",
		"Without a single database, atomicity spans services. Cover why two-phase commit blocks and scales poorly, the saga pattern with compensating actions (orchestration vs choreography), the outbox pattern for reliable event publishing, idempotent consumers, and eventual consistency as the pragmatic default.",
		content.DifficultyHard, 3.5},
	{"observability", "Observability: Logs, Metrics, Traces", "Understanding a system in production.",
		"Observability answers 'why is it slow/broken' from outputs. Cover structured logging and correlation IDs, the RED (rate, errors, duration) and USE methods for metrics, Prometheus counters/gauges/histograms and cardinality cost, and distributed tracing with spans and context propagation to follow a request across services.",
		content.DifficultyMedium, 3.0},
}

// seedBackendEngineeringTopics upserts the Backend Engineering pillar topics
// (dedup by track_id+slug). Returns slug→id for resource linking. Idempotent.
func (s *Seeder) seedBackendEngineeringTopics(tx *gorm.DB, trackID, pillarID uuid.UUID) (map[string]uuid.UUID, error) {
	out := make(map[string]uuid.UUID, len(backendEngineeringTopics))
	for i, d := range backendEngineeringTopics {
		hours := d.Hours
		if hours <= 0 {
			hours = 3.0
		}
		t := content.Topic{
			PillarID:       pillarID,
			TrackID:        trackID,
			Slug:           "be-" + d.Slug,
			Name:           d.Name,
			Summary:        ptr(d.Summary),
			ConceptMD:      ptr(d.Concept),
			Difficulty:     d.Diff,
			Priority:       content.Priority("high"),
			EstimatedHours: hours,
			SortOrder:      i,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "track_id"}, {Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"pillar_id", "name", "summary", "concept_md", "difficulty", "priority", "estimated_hours", "sort_order", "updated_at"}),
		}).Create(&t).Error; err != nil {
			return nil, err
		}
		var got content.Topic
		if err := tx.Where("track_id = ? AND slug = ?", trackID, t.Slug).First(&got).Error; err != nil {
			return nil, err
		}
		out[d.Slug] = got.ID
	}
	return out, nil
}
