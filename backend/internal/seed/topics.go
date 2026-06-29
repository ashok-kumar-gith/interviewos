package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/interviewos/backend/internal/content"
)

// patternDef defines a canonical DSA pattern. The slug is the natural key.
type patternDef struct {
	Slug      string
	Name      string
	Desc      string
	WhenToUse string
}

// canonicalPatterns is the set of interview DSA patterns. Topics in the DSA
// pillar map 1:1 to these patterns (slug-aligned).
var canonicalPatterns = []patternDef{
	{"arrays-hashing", "Arrays & Hashing", "Frequency maps, prefix sums, and set membership over arrays.", "When you need O(1) lookups or to count/deduplicate elements."},
	{"two-pointers", "Two Pointers", "Converging or parallel indices over a sequence.", "Sorted arrays, pair-sum, palindrome, in-place partitioning."},
	{"sliding-window", "Sliding Window", "A moving sub-range maintaining an invariant.", "Longest/shortest contiguous subarray or substring problems."},
	{"stack", "Stack", "LIFO processing, monotonic stacks.", "Matching/parsing, next-greater-element, expression evaluation."},
	{"binary-search", "Binary Search", "Halving a sorted/monotonic search space.", "Sorted arrays or 'minimize/maximize feasible value' problems."},
	{"linked-list", "Linked List", "Pointer manipulation over linked nodes.", "Reversal, cycle detection, merging, fast/slow pointers."},
	{"trees", "Trees", "DFS/BFS traversal and recursion over trees.", "Binary trees, BSTs, traversal and path problems."},
	{"tries", "Tries", "Prefix trees for string sets.", "Prefix search, autocomplete, word dictionaries."},
	{"heap", "Heap / Priority Queue", "Order statistics via a binary heap.", "Top-K, merge-K, streaming median, scheduling."},
	{"backtracking", "Backtracking", "Systematic search with undo.", "Permutations, combinations, subsets, constraint search."},
	{"graphs", "Graphs", "BFS/DFS, union-find, topological sort over graphs.", "Connectivity, shortest path, ordering, grids."},
	{"dynamic-programming", "Dynamic Programming", "Overlapping subproblems with memo/tabulation.", "Optimal substructure: knapsack, sequences, grids."},
	{"greedy", "Greedy", "Locally optimal choices that yield a global optimum.", "Interval scheduling, jump games, allocation."},
	{"intervals", "Intervals", "Sorting and merging ranges.", "Merge/insert intervals, meeting rooms, overlaps."},
	{"bit-manipulation", "Bit Manipulation", "Bitwise tricks and masks.", "XOR tricks, subsets via bitmask, single-number."},
	{"math", "Math & Geometry", "Number theory and matrix manipulation.", "GCD, primes, matrix rotation, pow/sqrt."},
}

// seedPatterns upserts the canonical patterns and returns slug→id.
func (s *Seeder) seedPatterns(tx *gorm.DB, trackID uuid.UUID) (map[string]uuid.UUID, error) {
	out := make(map[string]uuid.UUID, len(canonicalPatterns))
	for i, d := range canonicalPatterns {
		p := content.Pattern{
			TrackID:     trackID,
			Slug:        d.Slug,
			Name:        d.Name,
			Description: ptr(d.Desc),
			WhenToUse:   ptr(d.WhenToUse),
			SortOrder:   i,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"track_id", "name", "description", "when_to_use", "sort_order", "updated_at"}),
		}).Create(&p).Error; err != nil {
			return nil, err
		}
		var got content.Pattern
		if err := tx.Where("slug = ?", d.Slug).First(&got).Error; err != nil {
			return nil, err
		}
		out[d.Slug] = got.ID
	}
	return out, nil
}

// seedDSATopics creates one topic per pattern (slug-aligned) under the DSA
// pillar so problems can be grouped by topic and patterns. Returns slug→id.
func (s *Seeder) seedDSATopics(tx *gorm.DB, trackID, pillarID uuid.UUID) (map[string]uuid.UUID, error) {
	out := make(map[string]uuid.UUID, len(canonicalPatterns))
	for i, d := range canonicalPatterns {
		diff := content.DifficultyMedium
		t := content.Topic{
			PillarID:       pillarID,
			TrackID:        trackID,
			Slug:           "dsa-" + d.Slug,
			Name:           d.Name,
			Summary:        ptr(d.Desc),
			Difficulty:     diff,
			Priority:       content.Priority("high"),
			EstimatedHours: 4.0,
			SortOrder:      i,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "track_id"}, {Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"pillar_id", "name", "summary", "difficulty", "priority", "estimated_hours", "sort_order", "updated_at"}),
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

// sdTopicDef defines a System Design topic with theory and a difficulty.
type sdTopicDef struct {
	Slug    string
	Name    string
	Summary string
	Concept string
	Diff    content.Difficulty
}

var systemDesignTopics = []sdTopicDef{
	{"caching", "Caching", "Cache strategies, eviction, and invalidation.", "Covers cache-aside, write-through, write-back; LRU/LFU eviction; TTLs; and the cache-invalidation problem. Discuss Redis/Memcached and multi-tier caching.", content.DifficultyMedium},
	{"load-balancing", "Load Balancing", "Distributing traffic across servers.", "L4 vs L7 load balancing; algorithms (round-robin, least-connections, consistent hashing); health checks; sticky sessions; and global vs local balancing.", content.DifficultyMedium},
	{"sharding-partitioning", "Sharding & Partitioning", "Horizontal data partitioning.", "Range vs hash vs directory partitioning; hot-spotting; resharding; and the trade-offs of co-locating related data.", content.DifficultyHard},
	{"replication", "Replication", "Copying data across nodes.", "Single-leader, multi-leader, and leaderless replication; sync vs async; replication lag; and read-your-writes consistency.", content.DifficultyHard},
	{"cap-theorem", "CAP & Consistency", "Trade-offs under partition.", "CAP and PACELC; strong vs eventual consistency; quorum reads/writes; and tunable consistency models.", content.DifficultyMedium},
	{"consistent-hashing", "Consistent Hashing", "Stable key distribution under churn.", "Hash ring, virtual nodes, and minimal key movement when nodes join/leave. Used by caches, sharded stores, and load balancers.", content.DifficultyMedium},
	{"message-queues", "Message Queues", "Asynchronous, decoupled processing.", "Queue vs pub/sub; at-least-once vs exactly-once; ordering; backpressure; and dead-letter queues. Kafka, RabbitMQ, SQS.", content.DifficultyMedium},
	{"rate-limiting", "Rate Limiting", "Throttling request rates.", "Token bucket, leaky bucket, fixed/sliding window counters; distributed rate limiting with Redis; and 429 responses.", content.DifficultyMedium},
	{"cdn", "CDN", "Edge content delivery.", "Edge caching, origin pull vs push, cache invalidation, and geo-routing for static and dynamic content.", content.DifficultyEasy},
	{"db-indexing", "Database Indexing", "Speeding up reads with indexes.", "B-tree vs hash vs LSM indexes; covering and composite indexes; write amplification; and index selection trade-offs.", content.DifficultyMedium},
	{"consensus", "Consensus", "Agreement across distributed nodes.", "Leader election; Raft and Paxos at a high level; quorum; and where consensus is used (config, metadata, locks).", content.DifficultyHard},
	{"api-design", "API Design", "Designing robust service interfaces.", "REST vs gRPC vs GraphQL; idempotency; pagination; versioning; and error contracts.", content.DifficultyEasy},
}

// seedSystemDesignTopics upserts SD topics under the System Design pillar.
func (s *Seeder) seedSystemDesignTopics(tx *gorm.DB, trackID, pillarID uuid.UUID) (map[string]uuid.UUID, error) {
	out := make(map[string]uuid.UUID, len(systemDesignTopics))
	for i, d := range systemDesignTopics {
		t := content.Topic{
			PillarID:       pillarID,
			TrackID:        trackID,
			Slug:           "sd-" + d.Slug,
			Name:           d.Name,
			Summary:        ptr(d.Summary),
			ConceptMD:      ptr(d.Concept),
			Difficulty:     d.Diff,
			Priority:       content.Priority("high"),
			EstimatedHours: 3.0,
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
