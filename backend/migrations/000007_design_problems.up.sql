-- 000007_design_problems.up.sql
-- Design Problems (HLD / System Design) module per docs/04-DATABASE-SCHEMA.md §4.2.
-- Table: design_problems (structured markdown sections + JSONB section map).
--
-- This table is seeded/migration-managed. It carries deleted_at so content can
-- be retired without breaking FK history (mock_interviews.design_problem_id
-- references it with ON DELETE SET NULL).
--
-- Depends on 000001 (gen_random_uuid, set_updated_at) and 000002 (tracks,
-- difficulty enum). The backend-sde3 track row is ensured idempotently here so
-- the design_problems seed is self-sufficient on a fresh database.

BEGIN;

-- design_problems -----------------------------------------------------------
CREATE TABLE IF NOT EXISTS design_problems (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id               UUID NOT NULL REFERENCES tracks (id) ON DELETE RESTRICT,
    pillar_id              UUID NULL REFERENCES pillars (id) ON DELETE SET NULL,
    slug                   TEXT NOT NULL,
    title                  TEXT NOT NULL,
    difficulty             difficulty NOT NULL,
    order_index            INTEGER NOT NULL DEFAULT 0,
    requirements_md        TEXT NULL,
    capacity_estimation_md TEXT NULL,
    api_design_md          TEXT NULL,
    data_model_md          TEXT NULL,
    high_level_design_md   TEXT NULL,
    caching_md             TEXT NULL,
    queueing_md            TEXT NULL,
    scaling_md             TEXT NULL,
    tradeoffs_md           TEXT NULL,
    failure_handling_md    TEXT NULL,
    alternatives_md        TEXT NULL,
    interview_tips_md      TEXT NULL,
    follow_up_questions    JSONB NOT NULL DEFAULT '[]',
    sections               JSONB NOT NULL DEFAULT '{}',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at             TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_design_problems_slug ON design_problems (slug);
CREATE INDEX IF NOT EXISTS idx_design_order ON design_problems (track_id, order_index);

DROP TRIGGER IF EXISTS trg_design_problems_updated_at ON design_problems;
CREATE TRIGGER trg_design_problems_updated_at BEFORE UPDATE ON design_problems
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Ensure the backend-sde3 track exists so the seed below resolves a track_id on
-- a fresh database (the Go seeder also creates it; this is idempotent).
INSERT INTO tracks (slug, name, description, seniority, is_active, sort_order)
VALUES ('backend-sde3', 'Backend SDE3',
        'Senior backend engineering interview preparation track.', 'SDE3', true, 0)
ON CONFLICT (slug) DO NOTHING;

-- Seed: ordered HLD catalog (URL Shortener -> Twitter). Idempotent.
INSERT INTO design_problems
    (track_id, slug, title, difficulty, order_index,
     requirements_md, capacity_estimation_md, api_design_md, data_model_md,
     high_level_design_md, caching_md, queueing_md, scaling_md, tradeoffs_md,
     failure_handling_md, alternatives_md, interview_tips_md, follow_up_questions)
VALUES
((SELECT id FROM tracks WHERE slug = 'backend-sde3'),
 'url-shortener', 'Design a URL Shortener', 'easy', 1,
 'Functional: shorten a long URL into a unique short alias, redirect a short alias back to the original, and optionally support custom aliases and expiry. Non-functional: very high read-to-write ratio (~100:1), low-latency redirects (<100ms p99), high availability for reads, and short codes that are unguessable enough to avoid trivial enumeration.',
 'Assume 100M new URLs/month (~40 writes/s) and 10B redirects/month (~4K reads/s). At 500 bytes per record that is ~50GB/year of metadata. A 7-character base62 key space (62^7 ≈ 3.5 trillion) comfortably covers decades of growth.',
 'POST /urls {long_url, custom_alias?, expires_at?} -> {short_url}. GET /{code} -> 301/302 redirect to the long URL. DELETE /urls/{code} to remove. Reads are cacheable; writes go through the application tier.',
 'urls(code PK, long_url, user_id, created_at, expires_at, click_count). The short code is the primary key. A counter or hash plus base62 encoding generates codes; uniqueness is enforced by the PK.',
 'A stateless API tier behind a load balancer writes new mappings to a partitioned key-value/SQL store and serves redirects. A key-generation service (pre-allocated counter ranges or a KGS) avoids collisions without a read-before-write.',
 'Redirects are read-heavy and immutable, so cache hot codes in Redis/CDN with a high TTL. An 80/20 access pattern means a modest cache absorbs most traffic and keeps the datastore load low.',
 'Click analytics are written asynchronously: the redirect path emits an event to a queue (Kafka) consumed by an analytics pipeline so the hot redirect path never blocks on analytics writes.',
 'Scale reads with cache + read replicas and a CDN at the edge. Shard the datastore by code prefix or hash. The KGS hands out disjoint counter ranges per app node so code generation scales horizontally.',
 'Counter-based codes are sequential and enumerable but collision-free; hash-based codes are unguessable but need collision handling. 301 (permanent) caches aggressively but loses per-click analytics; 302 (temporary) preserves analytics at higher origin load.',
 'If the datastore is briefly unavailable, serve redirects from cache and queue writes. Replicate across AZs; a failed KGS node only loses its unused range, never causing duplicate codes.',
 'Could use a hash (MD5/SHA truncated) instead of a counter, or a distributed ID generator like Snowflake. A pure CDN-edge KV store (Cloudflare Workers KV) trades consistency for latency.',
 'Lead with the read/write asymmetry — that single observation drives caching, CDN, and replica decisions. Be ready to defend your key-generation strategy against collisions and enumeration.',
 '["How would you support custom vanity aliases without collisions?", "How do you expire and garbage-collect old URLs?", "How would you add per-link click analytics at scale?"]'),

((SELECT id FROM tracks WHERE slug = 'backend-sde3'),
 'pastebin', 'Design Pastebin', 'easy', 2,
 'Functional: users paste text/code and get a shareable link; pastes can be public or unlisted, optionally expire, and support syntax highlighting. Non-functional: durable storage of large text blobs, fast reads, and read-heavy access similar to a URL shortener but with much larger payloads.',
 'Assume 10M new pastes/month with an average size of 10KB (some up to several MB). That is ~100GB/month of new blob data and a read:write ratio around 20:1. Metadata is small; the bulk of storage is the paste body.',
 'POST /pastes {content, expires_at?, visibility} -> {paste_id, url}. GET /pastes/{id} -> content + metadata. DELETE /pastes/{id}. Large bodies are stored in object storage, not the metadata database.',
 'pastes(id PK, blob_key, user_id, visibility, created_at, expires_at, size_bytes). The body lives in object storage (S3) keyed by blob_key; the database holds only metadata and the pointer.',
 'The API tier writes metadata to a database and the body to object storage, then returns an id. Reads fetch metadata, then stream the body from object storage (or a CDN) to the client.',
 'Cache hot paste metadata and small bodies in Redis; front object storage with a CDN so popular pastes are served from the edge. Immutable bodies make caching trivial with long TTLs.',
 'An expiry/cleanup job runs off a queue: TTL events enqueue deletions processed asynchronously so the write path is not blocked by garbage collection of object-storage blobs.',
 'Separate the metadata store (shardable SQL/NoSQL) from blob storage (horizontally scalable object store + CDN). This separation lets each scale independently along its own bottleneck.',
 'Storing blobs in the database simplifies transactions but bloats it and hurts cache locality; object storage scales better but adds a second round trip. Inline small bodies, externalize large ones.',
 'Object storage gives 11-nines durability; the metadata DB is replicated across AZs. If the blob is missing but metadata exists, return a clear 410 rather than a generic error.',
 'Could store everything in a wide-column store (Cassandra) to avoid a separate object store, or use a document DB for small pastes. A pure CDN+KV design works for size-capped pastes.',
 'Contrast this with the URL shortener to show you understand when payload size changes the design (object storage + CDN). Mention the size threshold for inlining vs externalizing bodies.',
 '["How do you handle very large (multi-MB) pastes efficiently?", "How would you implement private/expiring pastes securely?", "How do you prevent abuse such as spam or malware pastes?"]'),

((SELECT id FROM tracks WHERE slug = 'backend-sde3'),
 'notification-system', 'Design a Notification System', 'medium', 3,
 'Functional: send notifications across multiple channels (push, SMS, email, in-app), support templating, user preferences/opt-out, and scheduled/triggered sends. Non-functional: high throughput, at-least-once delivery, retries with backoff, deduplication, and rate limiting per user and per provider.',
 'Assume 50M users and 500M notifications/day (~6K/s average, with bursts to 50K/s). Each provider (APNs, FCM, Twilio, SES) has its own rate limits, so the system must shape and buffer traffic per provider.',
 'POST /notifications {user_id, template_id, channel, payload, send_at?}. Internally a dispatcher resolves preferences, renders the template, and enqueues per-channel sends. Webhooks ingest delivery receipts from providers.',
 'notifications(id, user_id, channel, status, template_id, created_at, sent_at), user_preferences(user_id, channel, enabled), templates(id, channel, body). Status transitions (queued->sent->delivered/failed) are tracked for observability.',
 'A producer API validates and writes the request, then publishes to a per-channel message queue. Channel workers pull messages, render templates, call the provider, and update status. Preference and rate-limit checks happen before dispatch.',
 'Cache user preferences and rendered templates so the hot dispatch path avoids repeated DB reads. A short-TTL dedup cache (idempotency key) prevents duplicate sends on retries.',
 'A message queue (Kafka/SQS) is the backbone: it decouples producers from channel workers, absorbs bursts, enables per-provider rate shaping, and supports retry queues with exponential backoff and a dead-letter queue.',
 'Scale horizontally by adding channel workers and partitioning queues by channel/provider. Per-provider token-bucket rate limiters protect against provider throttling. Sharding by user_id distributes preference lookups.',
 'At-least-once delivery plus dedup is simpler than exactly-once but can still double-send on rare races; exactly-once needs idempotency keys end to end. Synchronous sends give instant feedback but cannot absorb bursts the way queues can.',
 'Failed sends go to a retry queue with backoff, then a DLQ after N attempts for manual inspection. Provider outages are isolated per channel so one failing provider never blocks others.',
 'Could use a managed service (SNS, OneSignal) instead of building channel workers, or a single unified queue with routing keys instead of per-channel queues. Push providers themselves can fan out to reduce our load.',
 'Emphasize the queue as the central decoupling and burst-absorption mechanism, and call out idempotency/dedup early — interviewers probe how you avoid double-sending on retries.',
 '["How do you guarantee a user is not notified twice for the same event?", "How would you support user-level quiet hours and rate limits?", "How do you handle a provider (e.g. APNs) outage gracefully?"]'),

((SELECT id FROM tracks WHERE slug = 'backend-sde3'),
 'whatsapp', 'Design WhatsApp / a Chat System', 'hard', 4,
 'Functional: 1:1 and group messaging, online/last-seen presence, delivery and read receipts, message ordering, media sharing, and offline delivery. Non-functional: low end-to-end latency, durability of undelivered messages, horizontal scale to billions of devices, and optional end-to-end encryption.',
 'Assume 2B users and 100B messages/day (~1.2M messages/s). Persistent connections dominate: tens of millions of concurrent WebSocket connections require many connection-gateway servers and a way to route a message to the server holding the recipient connection.',
 'WebSocket for real-time send/receive. REST for history: GET /conversations/{id}/messages. Internally: send(message) -> persist -> route to recipient gateway -> push, or store-and-forward if the recipient is offline.',
 'messages(id, conversation_id, sender_id, seq, body, created_at), conversations(id, type), conversation_members(conversation_id, user_id). A monotonic per-conversation sequence number provides ordering; receipts are a separate status table.',
 'Clients hold a persistent WebSocket to a connection gateway. A session registry maps user_id -> gateway. On send, the message service persists the message, looks up the recipient gateway, and forwards it; offline recipients get store-and-forward delivery on reconnect.',
 'Cache the session registry (user->gateway) and recent conversation messages in Redis for fast routing and history reads. Presence is a short-TTL cache key refreshed by heartbeats.',
 'A message queue/bus delivers messages between gateways and to offline-storage and fan-out workers. Group sends fan out via the queue so the sender path stays fast regardless of group size.',
 'Shard by conversation_id (keeps a conversation co-located and ordered) or user_id. Connection gateways scale horizontally behind a layer-4 load balancer; the session registry is partitioned and replicated.',
 'Per-conversation sequencing gives strong ordering but limits a conversation to one shard; global ordering does not scale. Store-and-forward guarantees delivery at the cost of storage; fire-and-forget is cheaper but loses offline messages.',
 'Undelivered messages persist until acked, so a gateway crash only drops live connections (clients reconnect and resync from last seq). Replicate message storage across AZs; receipts are eventually consistent.',
 'Could use XMPP/MQTT instead of raw WebSockets, or a managed pub/sub (Pusher/Ably). Long polling is a fallback for restricted networks. E2E encryption (Signal protocol) moves key management to clients.',
 'Spend your time on the connection-gateway + session-registry routing problem and on message ordering — those are the parts that distinguish chat from a generic CRUD system. Mention store-and-forward for offline users.',
 '["How do you guarantee message ordering within a conversation?", "How do you route a message to the right server among millions of connections?", "How would you scale group messages to very large groups?"]'),

((SELECT id FROM tracks WHERE slug = 'backend-sde3'),
 'instagram', 'Design Instagram', 'hard', 5,
 'Functional: upload photos/videos, follow users, view a personalized home feed, like and comment, and view user profiles. Non-functional: read-heavy feed serving at low latency, durable media storage, and a feed that stays fresh as new posts arrive.',
 'Assume 500M daily active users, 100M photo uploads/day, and feed reads orders of magnitude higher. Average photo ~2MB after compression gives ~200TB/day of new media, so media storage and CDN delivery dominate cost.',
 'POST /posts (multipart media + caption). GET /feed?cursor= returns a paginated, ranked list of posts from followees. POST /posts/{id}/like, POST /posts/{id}/comments. Media uploads go to object storage via pre-signed URLs.',
 'posts(id, user_id, media_key, caption, created_at), follows(follower_id, followee_id), feed_cache(user_id, post_id, score). Media bodies live in object storage + CDN; metadata and the social graph live in sharded databases.',
 'Upload writes media to object storage and metadata to a posts store, then triggers feed fan-out. Reads serve a precomputed feed from a feed store, hydrating post metadata and media URLs (CDN) per item.',
 'Aggressively cache feeds, post metadata, and counts (likes/comments) in Redis; serve all media through a CDN. Hot timelines live fully in cache to keep feed reads in single-digit milliseconds.',
 'New posts publish to a queue that fan-out workers consume to push the post id into each follower''s feed (fan-out on write). The queue smooths upload spikes and decouples posting from feed materialization.',
 'Shard posts and follows by user_id; replicate read paths. Fan-out on write precomputes feeds for fast reads; celebrity accounts with millions of followers use fan-out on read (pull) to avoid write amplification — a hybrid model.',
 'Fan-out on write gives fast reads but expensive writes and storage; fan-out on read is cheap to write but slow to read. The hybrid (push for normal users, pull for celebrities) balances both at the cost of complexity.',
 'Media in object storage is highly durable; the feed cache is reconstructable from source data, so cache loss degrades latency, not correctness. Cross-AZ replication protects metadata and the social graph.',
 'Could use a pure pull model with strong ranking at read time, or a graph database for the social graph. A managed feed service or timeline-as-a-service abstracts fan-out. ML ranking can replace chronological ordering.',
 'The fan-out write-vs-read trade-off and the celebrity (hot-key) problem are the core of this question — present the hybrid model explicitly. Note that media+CDN dominate storage and bandwidth.',
 '["How do you handle celebrities with tens of millions of followers (the hot-key problem)?", "How do you keep the feed fresh and ranked?", "How would you store and serve media efficiently worldwide?"]'),

((SELECT id FROM tracks WHERE slug = 'backend-sde3'),
 'twitter', 'Design Twitter', 'hard', 6,
 'Functional: post tweets, follow users, view a home timeline (tweets from followees) and a user timeline, like/retweet, and search. Non-functional: extremely read-heavy timeline serving at low latency, real-time-ish freshness, and graceful handling of celebrity accounts with massive follower counts.',
 'Assume 300M daily active users, 500M tweets/day (~6K writes/s), and timeline reads on the order of 100x higher. Tweets are small (~300 bytes) so the social graph and timeline materialization, not raw storage, are the scaling challenge.',
 'POST /tweets {text, media?}. GET /timeline/home?cursor= (followees'' tweets, paginated). GET /timeline/user/{id}. POST /tweets/{id}/retweet, /like. Search via a separate indexed service.',
 'tweets(id, user_id, text, created_at), follows(follower_id, followee_id), home_timeline(user_id, tweet_id, score). Tweet ids are time-sortable (Snowflake) so timelines order naturally and cursors page cleanly.',
 'On post, the tweet is written to the tweet store and fanned out to followers'' home timelines. Timeline reads serve a precomputed list from a fast timeline store, hydrating tweet content from cache. Search indexes tweets asynchronously.',
 'Home timelines and tweet bodies live in Redis for millisecond reads; counts (likes/retweets) are cached and updated asynchronously. The vast read volume makes caching the primary scaling lever.',
 'A fan-out queue distributes each new tweet to follower timelines (fan-out on write). The queue absorbs posting spikes and lets fan-out scale independently; the same bus feeds the search indexer and analytics.',
 'Shard tweets and the social graph by user_id; replicate timeline reads widely. Fan-out on write for normal users plus fan-out on read (merge at query time) for celebrities is the standard hybrid to avoid write amplification.',
 'Push (fan-out on write) optimizes the common read path but blows up for celebrities; pull (fan-out on read) is cheap to write but costly to read. The hybrid model accepts extra complexity to bound both. Time-sortable ids avoid a global sort.',
 'Timelines are reconstructable from tweets + follows, so a cache outage degrades latency, not data. Tweet storage and the social graph are replicated across AZs; the indexer can replay from the tweet log after a failure.',
 'Could rank timelines with ML instead of reverse-chronological, use a graph DB for follows, or a unified log (Kafka) as the source of truth with materialized views. Search can be Elasticsearch fed off the tweet stream.',
 'This is the canonical fan-out question — be explicit about push vs pull and the celebrity hybrid, and mention time-sortable Snowflake ids for ordering and pagination. Distinguish it from Instagram by the smaller payload and search emphasis.',
 '["How do you handle the celebrity fan-out problem?", "How would you implement tweet search at scale?", "How do you keep like/retweet counts accurate yet performant under heavy concurrency?"]');

COMMIT;
