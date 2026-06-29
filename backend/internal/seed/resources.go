package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/interviewos/backend/internal/content"
)

// resourceDef defines a deduplicated global resource. Slug is the in-seed key;
// url is the DB-level dedup key (unique where present).
type resourceDef struct {
	Slug     string
	Type     content.ResourceType
	Title    string
	Author   string
	URL      string
	Provider string
	Desc     string
	Minutes  int
}

// globalResources are deduplicated by url at the DB layer.
var globalResources = []resourceDef{
	{"ddia", "book", "Designing Data-Intensive Applications", "Martin Kleppmann", "https://dataintensive.net/", "O'Reilly", "The canonical reference for distributed data systems: replication, partitioning, consistency, and stream processing.", 2400},
	{"system-design-primer", "github", "The System Design Primer", "Donne Martin", "https://github.com/donnemartin/system-design-primer", "GitHub", "Open-source study guide for system design interviews with worked examples.", 1200},
	{"neetcode-roadmap", "video", "NeetCode DSA Roadmap & Pattern Videos", "NeetCode", "https://neetcode.io/roadmap", "NeetCode", "Pattern-organized video walkthroughs for the canonical DSA problem set.", 1800},
	{"grokking-coding-patterns", "course", "Grokking the Coding Interview: Patterns", "DesignGurus", "https://www.designgurus.io/course/grokking-the-coding-interview", "DesignGurus", "Pattern-first coding-interview course mapping problems to reusable approaches.", 1500},
	{"grokking-system-design", "course", "Grokking the System Design Interview", "DesignGurus", "https://www.designgurus.io/course/grokking-the-system-design-interview", "DesignGurus", "Structured walkthroughs of common system-design interview questions.", 1200},
	{"cracking-coding-interview", "book", "Cracking the Coding Interview", "Gayle Laakmann McDowell", "https://www.crackingthecodinginterview.com/", "CareerCup", "Classic interview-prep book covering DSA, behavioral, and problem-solving strategy.", 1800},
	{"redis-caching-docs", "documentation", "Redis Caching Patterns", "Redis", "https://redis.io/docs/latest/develop/use/patterns/", "Redis", "Official documentation on caching strategies and data structures.", 60},
	{"kafka-docs", "documentation", "Apache Kafka Documentation", "Apache", "https://kafka.apache.org/documentation/", "Apache", "Reference for message-queue and event-streaming concepts.", 120},
	{"raft-paper", "article", "In Search of an Understandable Consensus Algorithm (Raft)", "Diego Ongaro, John Ousterhout", "https://raft.github.io/raft.pdf", "raft.github.io", "The Raft consensus paper, explaining leader election and log replication.", 90},
	{"use-the-index-luke", "article", "Use The Index, Luke", "Markus Winand", "https://use-the-index-luke.com/", "use-the-index-luke.com", "A practical guide to SQL indexing and query performance.", 120},
	// Backend Engineering depth resources (deduped by url).
	{"the-go-programming-language", "book", "The Go Programming Language", "Alan Donovan, Brian Kernighan", "https://www.gopl.io/", "Addison-Wesley", "The definitive Go book covering goroutines, channels, the runtime, and idiomatic concurrency.", 1800},
	{"systems-performance-gregg", "book", "Systems Performance", "Brendan Gregg", "https://www.brendangregg.com/systems-performance-2nd-edition-book.html", "Pearson", "The reference on performance methodology and profiling: CPU, memory, IO, and tools like perf, BPF, and flame graphs.", 2000},
	{"postgres-mvcc-docs", "documentation", "PostgreSQL: Concurrency Control & MVCC", "PostgreSQL", "https://www.postgresql.org/docs/current/mvcc.html", "PostgreSQL", "Official docs on transaction isolation, MVCC visibility, and locking in Postgres.", 90},
	{"raft-visualization", "article", "The Raft Consensus Visualization", "Diego Ongaro et al.", "https://raft.github.io/", "raft.github.io", "Interactive explanation of leader election and log replication in Raft.", 60},
	{"kubernetes-docs-concepts", "documentation", "Kubernetes Concepts", "Kubernetes", "https://kubernetes.io/docs/concepts/", "Kubernetes", "Official reference for Pods, Deployments, Services, scheduling, and autoscaling.", 180},
	{"docker-docs", "documentation", "Docker Documentation", "Docker", "https://docs.docker.com/get-started/", "Docker", "Official guide to images, layering, multi-stage builds, networking, and volumes.", 120},
	{"high-performance-browser-networking", "book", "High Performance Browser Networking", "Ilya Grigorik", "https://hpbn.co/", "O'Reilly", "Free online book on TCP, TLS, HTTP/2, HTTP/3 and the performance characteristics of the network stack.", 900},
	{"go-memory-model", "documentation", "The Go Memory Model", "Go Team", "https://go.dev/ref/mem", "go.dev", "The happens-before rules governing concurrent goroutine memory access.", 45},
	{"nginx-admin-guide", "documentation", "NGINX Admin Guide", "F5/NGINX", "https://docs.nginx.com/nginx/admin-guide/", "NGINX", "Reverse proxy, load-balancing, upstreams, and TLS termination configuration.", 120},
}

// seedResources upserts global resources (dedup by url) and returns slug→id.
func (s *Seeder) seedResources(tx *gorm.DB) (map[string]content.Resource, error) {
	out := make(map[string]content.Resource, len(globalResources))
	for _, d := range globalResources {
		r := content.Resource{
			Type:             d.Type,
			Title:            d.Title,
			Author:           ptr(d.Author),
			URL:              ptr(d.URL),
			Provider:         ptr(d.Provider),
			Description:      ptr(d.Desc),
			EstimatedMinutes: ptr(d.Minutes),
			Priority:         content.Priority("high"),
			IsFree:           true,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "url"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Expr{SQL: "url IS NOT NULL"},
			}},
			DoUpdates: clause.AssignmentColumns([]string{"type", "title", "author", "provider", "description", "estimated_minutes", "priority", "updated_at"}),
		}).Create(&r).Error; err != nil {
			return nil, err
		}
		var got content.Resource
		if err := tx.Where("url = ?", d.URL).First(&got).Error; err != nil {
			return nil, err
		}
		out[d.Slug] = got
	}
	return out, nil
}

// topicResourceLink maps a topic slug (key in the dsa/sd maps) to resource slugs.
type topicResourceLink struct {
	topicSlug     string
	resourceSlugs []string
}

// dsaTopicResources links DSA pattern topics to resources (keyed by pattern slug).
var dsaTopicResources = []topicResourceLink{
	{"arrays-hashing", []string{"neetcode-roadmap", "grokking-coding-patterns", "cracking-coding-interview"}},
	{"two-pointers", []string{"neetcode-roadmap", "grokking-coding-patterns"}},
	{"sliding-window", []string{"neetcode-roadmap", "grokking-coding-patterns"}},
	{"binary-search", []string{"neetcode-roadmap", "cracking-coding-interview"}},
	{"trees", []string{"neetcode-roadmap", "cracking-coding-interview"}},
	{"graphs", []string{"neetcode-roadmap", "grokking-coding-patterns"}},
	{"dynamic-programming", []string{"neetcode-roadmap", "grokking-coding-patterns", "cracking-coding-interview"}},
	{"heap", []string{"neetcode-roadmap"}},
}

// sdTopicResources links System Design topics to resources (keyed by SD slug).
var sdTopicResources = []topicResourceLink{
	{"caching", []string{"system-design-primer", "redis-caching-docs", "grokking-system-design"}},
	{"load-balancing", []string{"system-design-primer", "grokking-system-design"}},
	{"sharding-partitioning", []string{"ddia", "grokking-system-design"}},
	{"replication", []string{"ddia", "grokking-system-design"}},
	{"cap-theorem", []string{"ddia", "system-design-primer"}},
	{"consistent-hashing", []string{"system-design-primer"}},
	{"message-queues", []string{"kafka-docs", "ddia"}},
	{"rate-limiting", []string{"system-design-primer"}},
	{"cdn", []string{"system-design-primer"}},
	{"db-indexing", []string{"use-the-index-luke", "ddia"}},
	{"consensus", []string{"raft-paper", "ddia"}},
	{"api-design", []string{"system-design-primer"}},
}

// beTopicResources links Backend Engineering topics to resources (keyed by the
// be-topic slug, without the "be-" prefix). Resources are reused where they
// already exist (ddia, kafka-docs, redis-caching-docs, use-the-index-luke,
// raft-paper, system-design-primer) and deduped by url at the DB layer.
var beTopicResources = []topicResourceLink{
	{"kafka-event-streaming", []string{"kafka-docs", "ddia"}},
	{"redis-caching-patterns", []string{"redis-caching-docs", "ddia"}},
	{"sql-query-optimization", []string{"use-the-index-luke", "postgres-mvcc-docs"}},
	{"nosql-data-models", []string{"ddia", "system-design-primer"}},
	{"transactions-isolation", []string{"ddia", "postgres-mvcc-docs"}},
	{"mvcc", []string{"postgres-mvcc-docs", "ddia"}},
	{"indexing-btree-lsm", []string{"ddia", "use-the-index-luke"}},
	{"replication", []string{"ddia"}},
	{"partitioning-sharding", []string{"ddia", "system-design-primer"}},
	{"cap-pacelc", []string{"ddia", "system-design-primer"}},
	{"consensus-raft-paxos", []string{"raft-paper", "raft-visualization", "ddia"}},
	{"load-balancing-nginx", []string{"nginx-admin-guide", "system-design-primer"}},
	{"cdn", []string{"system-design-primer"}},
	{"docker-containers", []string{"docker-docs"}},
	{"kubernetes", []string{"kubernetes-docs-concepts", "docker-docs"}},
	{"networking-tcp-tls-http", []string{"high-performance-browser-networking"}},
	{"linux-fundamentals", []string{"systems-performance-gregg"}},
	{"concurrency-synchronization", []string{"the-go-programming-language", "go-memory-model"}},
	{"memory-performance-profiling", []string{"systems-performance-gregg"}},
	{"go-runtime", []string{"the-go-programming-language", "go-memory-model"}},
	{"garbage-collection", []string{"the-go-programming-language"}},
	{"api-design-idempotency", []string{"system-design-primer"}},
	{"distributed-transactions-saga", []string{"ddia"}},
	{"observability", []string{"systems-performance-gregg"}},
}

// seedTopicResources links topics to resources, deduped by (topic_id, resource_id).
func (s *Seeder) seedTopicResources(tx *gorm.DB, dsaTopics, sdTopics, beTopics map[string]uuid.UUID, resources map[string]content.Resource) error {
	link := func(topicID uuid.UUID, resourceSlugs []string) error {
		for i, rs := range resourceSlugs {
			res, ok := resources[rs]
			if !ok {
				continue
			}
			tr := content.TopicResource{
				TopicID:    topicID,
				ResourceID: res.ID,
				Relevance:  content.Priority("high"),
				IsPrimary:  i == 0,
				SortOrder:  i,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "topic_id"}, {Name: "resource_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"relevance", "is_primary", "sort_order", "updated_at"}),
			}).Create(&tr).Error; err != nil {
				return err
			}
		}
		return nil
	}
	for _, l := range dsaTopicResources {
		if id, ok := dsaTopics[l.topicSlug]; ok {
			if err := link(id, l.resourceSlugs); err != nil {
				return err
			}
		}
	}
	for _, l := range sdTopicResources {
		if id, ok := sdTopics[l.topicSlug]; ok {
			if err := link(id, l.resourceSlugs); err != nil {
				return err
			}
		}
	}
	for _, l := range beTopicResources {
		if id, ok := beTopics[l.topicSlug]; ok {
			if err := link(id, l.resourceSlugs); err != nil {
				return err
			}
		}
	}
	return nil
}
