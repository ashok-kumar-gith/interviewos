package seed

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/interviewos/backend/internal/content"
)

// problemDef is one canonical (deduplicated) DSA problem. Slug is the canonical
// natural key; the same problem appearing on multiple curated lists is recorded
// once with multiple Sources entries. PatternSlugs maps to seeded patterns; the
// first pattern also determines the topic grouping.
type problemDef struct {
	Slug         string
	Title        string
	LeetCodeID   string
	Difficulty   content.Difficulty
	PatternSlugs []string
	Sources      []content.ProblemSourceName
	Frequency    float64
}

const lcBase = "https://leetcode.com/problems/"

// premiumProblems is the set of canonical problem slugs that require a
// LeetCode Premium subscription. Kept separate so the positional problem
// literals below stay compact.
var premiumProblems = map[string]struct{}{
	"encode-and-decode-strings": {},
	"graph-valid-tree":          {},
	"alien-dictionary":          {},
	"meeting-rooms-ii":          {},
}

// canonicalProblems is the merged, deduplicated DSA set (Blind 75 / NeetCode 150
// / Grind 75). Each problem maps to >=1 pattern and records its source lists.
var canonicalProblems = []problemDef{
	// Arrays & Hashing
	{"two-sum", "Two Sum", "1", content.DifficultyEasy, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 95},
	{"contains-duplicate", "Contains Duplicate", "217", content.DifficultyEasy, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 70},
	{"valid-anagram", "Valid Anagram", "242", content.DifficultyEasy, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 65},
	{"group-anagrams", "Group Anagrams", "49", content.DifficultyMedium, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 78},
	{"top-k-frequent-elements", "Top K Frequent Elements", "347", content.DifficultyMedium, []string{"arrays-hashing", "heap"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 82},
	{"product-of-array-except-self", "Product of Array Except Self", "238", content.DifficultyMedium, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 80},
	{"valid-sudoku", "Valid Sudoku", "36", content.DifficultyMedium, []string{"arrays-hashing"}, []content.ProblemSourceName{"neetcode150"}, 40},
	{"longest-consecutive-sequence", "Longest Consecutive Sequence", "128", content.DifficultyMedium, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 60},
	{"encode-and-decode-strings", "Encode and Decode Strings", "271", content.DifficultyMedium, []string{"arrays-hashing"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 45},

	// Two Pointers
	{"valid-palindrome", "Valid Palindrome", "125", content.DifficultyEasy, []string{"two-pointers"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 68},
	{"two-sum-ii", "Two Sum II - Input Array Is Sorted", "167", content.DifficultyMedium, []string{"two-pointers"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 55},
	{"3sum", "3Sum", "15", content.DifficultyMedium, []string{"two-pointers"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 85},
	{"container-with-most-water", "Container With Most Water", "11", content.DifficultyMedium, []string{"two-pointers"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 72},
	{"trapping-rain-water", "Trapping Rain Water", "42", content.DifficultyHard, []string{"two-pointers", "stack"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 75},

	// Sliding Window
	{"best-time-to-buy-and-sell-stock", "Best Time to Buy and Sell Stock", "121", content.DifficultyEasy, []string{"sliding-window"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 80},
	{"longest-substring-without-repeating-characters", "Longest Substring Without Repeating Characters", "3", content.DifficultyMedium, []string{"sliding-window"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 88},
	{"longest-repeating-character-replacement", "Longest Repeating Character Replacement", "424", content.DifficultyMedium, []string{"sliding-window"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 50},
	{"minimum-window-substring", "Minimum Window Substring", "76", content.DifficultyHard, []string{"sliding-window"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 70},

	// Stack
	{"valid-parentheses", "Valid Parentheses", "20", content.DifficultyEasy, []string{"stack"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 84},
	{"min-stack", "Min Stack", "155", content.DifficultyMedium, []string{"stack"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 58},
	{"daily-temperatures", "Daily Temperatures", "739", content.DifficultyMedium, []string{"stack"}, []content.ProblemSourceName{"neetcode150"}, 52},
	{"car-fleet", "Car Fleet", "853", content.DifficultyMedium, []string{"stack"}, []content.ProblemSourceName{"neetcode150"}, 30},
	{"largest-rectangle-in-histogram", "Largest Rectangle in Histogram", "84", content.DifficultyHard, []string{"stack"}, []content.ProblemSourceName{"neetcode150"}, 48},

	// Binary Search
	{"binary-search", "Binary Search", "704", content.DifficultyEasy, []string{"binary-search"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 60},
	{"search-a-2d-matrix", "Search a 2D Matrix", "74", content.DifficultyMedium, []string{"binary-search"}, []content.ProblemSourceName{"neetcode150"}, 45},
	{"koko-eating-bananas", "Koko Eating Bananas", "875", content.DifficultyMedium, []string{"binary-search"}, []content.ProblemSourceName{"neetcode150"}, 50},
	{"find-minimum-in-rotated-sorted-array", "Find Minimum in Rotated Sorted Array", "153", content.DifficultyMedium, []string{"binary-search"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 62},
	{"search-in-rotated-sorted-array", "Search in Rotated Sorted Array", "33", content.DifficultyMedium, []string{"binary-search"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 74},
	{"median-of-two-sorted-arrays", "Median of Two Sorted Arrays", "4", content.DifficultyHard, []string{"binary-search"}, []content.ProblemSourceName{"neetcode150"}, 66},

	// Linked List
	{"reverse-linked-list", "Reverse Linked List", "206", content.DifficultyEasy, []string{"linked-list"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 82},
	{"merge-two-sorted-lists", "Merge Two Sorted Lists", "21", content.DifficultyEasy, []string{"linked-list"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 76},
	{"linked-list-cycle", "Linked List Cycle", "141", content.DifficultyEasy, []string{"linked-list"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 64},
	{"reorder-list", "Reorder List", "143", content.DifficultyMedium, []string{"linked-list"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 48},
	{"remove-nth-node-from-end-of-list", "Remove Nth Node From End of List", "19", content.DifficultyMedium, []string{"linked-list", "two-pointers"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 58},
	{"copy-list-with-random-pointer", "Copy List with Random Pointer", "138", content.DifficultyMedium, []string{"linked-list"}, []content.ProblemSourceName{"neetcode150"}, 50},
	{"merge-k-sorted-lists", "Merge k Sorted Lists", "23", content.DifficultyHard, []string{"linked-list", "heap"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 72},

	// Trees
	{"invert-binary-tree", "Invert Binary Tree", "226", content.DifficultyEasy, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 60},
	{"maximum-depth-of-binary-tree", "Maximum Depth of Binary Tree", "104", content.DifficultyEasy, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 58},
	{"diameter-of-binary-tree", "Diameter of Binary Tree", "543", content.DifficultyEasy, []string{"trees"}, []content.ProblemSourceName{"neetcode150"}, 40},
	{"same-tree", "Same Tree", "100", content.DifficultyEasy, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 42},
	{"subtree-of-another-tree", "Subtree of Another Tree", "572", content.DifficultyEasy, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 38},
	{"lowest-common-ancestor-of-a-binary-search-tree", "Lowest Common Ancestor of a BST", "235", content.DifficultyMedium, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 56},
	{"binary-tree-level-order-traversal", "Binary Tree Level Order Traversal", "102", content.DifficultyMedium, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 64},
	{"validate-binary-search-tree", "Validate Binary Search Tree", "98", content.DifficultyMedium, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 68},
	{"kth-smallest-element-in-a-bst", "Kth Smallest Element in a BST", "230", content.DifficultyMedium, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 46},
	{"construct-binary-tree-from-preorder-and-inorder-traversal", "Construct Binary Tree from Preorder and Inorder Traversal", "105", content.DifficultyMedium, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 50},
	{"binary-tree-maximum-path-sum", "Binary Tree Maximum Path Sum", "124", content.DifficultyHard, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 58},
	{"serialize-and-deserialize-binary-tree", "Serialize and Deserialize Binary Tree", "297", content.DifficultyHard, []string{"trees"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 60},

	// Tries
	{"implement-trie-prefix-tree", "Implement Trie (Prefix Tree)", "208", content.DifficultyMedium, []string{"tries"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 54},
	{"design-add-and-search-words-data-structure", "Design Add and Search Words Data Structure", "211", content.DifficultyMedium, []string{"tries"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 44},
	{"word-search-ii", "Word Search II", "212", content.DifficultyHard, []string{"tries", "backtracking"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 56},

	// Heap / Priority Queue
	{"kth-largest-element-in-a-stream", "Kth Largest Element in a Stream", "703", content.DifficultyEasy, []string{"heap"}, []content.ProblemSourceName{"neetcode150"}, 40},
	{"find-median-from-data-stream", "Find Median from Data Stream", "295", content.DifficultyHard, []string{"heap"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 58},
	{"task-scheduler", "Task Scheduler", "621", content.DifficultyMedium, []string{"heap", "greedy"}, []content.ProblemSourceName{"neetcode150"}, 50},

	// Backtracking
	{"subsets", "Subsets", "78", content.DifficultyMedium, []string{"backtracking"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 60},
	{"combination-sum", "Combination Sum", "39", content.DifficultyMedium, []string{"backtracking"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 64},
	{"permutations", "Permutations", "46", content.DifficultyMedium, []string{"backtracking"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 62},
	{"word-search", "Word Search", "79", content.DifficultyMedium, []string{"backtracking", "graphs"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 66},

	// Graphs
	{"number-of-islands", "Number of Islands", "200", content.DifficultyMedium, []string{"graphs"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 86},
	{"clone-graph", "Clone Graph", "133", content.DifficultyMedium, []string{"graphs"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 58},
	{"pacific-atlantic-water-flow", "Pacific Atlantic Water Flow", "417", content.DifficultyMedium, []string{"graphs"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 46},
	{"course-schedule", "Course Schedule", "207", content.DifficultyMedium, []string{"graphs"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 72},
	{"rotting-oranges", "Rotting Oranges", "994", content.DifficultyMedium, []string{"graphs"}, []content.ProblemSourceName{"grind75"}, 54},
	{"graph-valid-tree", "Graph Valid Tree", "261", content.DifficultyMedium, []string{"graphs"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 40},
	{"alien-dictionary", "Alien Dictionary", "269", content.DifficultyHard, []string{"graphs"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 50},
	{"word-ladder", "Word Ladder", "127", content.DifficultyHard, []string{"graphs"}, []content.ProblemSourceName{"neetcode150"}, 48},

	// Dynamic Programming
	{"climbing-stairs", "Climbing Stairs", "70", content.DifficultyEasy, []string{"dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 66},
	{"house-robber", "House Robber", "198", content.DifficultyMedium, []string{"dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 60},
	{"coin-change", "Coin Change", "322", content.DifficultyMedium, []string{"dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 70},
	{"longest-increasing-subsequence", "Longest Increasing Subsequence", "300", content.DifficultyMedium, []string{"dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 62},
	{"longest-common-subsequence", "Longest Common Subsequence", "1143", content.DifficultyMedium, []string{"dynamic-programming"}, []content.ProblemSourceName{"neetcode150"}, 58},
	{"word-break", "Word Break", "139", content.DifficultyMedium, []string{"dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 64},
	{"unique-paths", "Unique Paths", "62", content.DifficultyMedium, []string{"dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 52},
	{"maximum-subarray", "Maximum Subarray", "53", content.DifficultyMedium, []string{"dynamic-programming", "greedy"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 78},

	// Greedy
	{"jump-game", "Jump Game", "55", content.DifficultyMedium, []string{"greedy"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 60},

	// Intervals
	{"insert-interval", "Insert Interval", "57", content.DifficultyMedium, []string{"intervals"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 54},
	{"merge-intervals", "Merge Intervals", "56", content.DifficultyMedium, []string{"intervals"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 80},
	{"non-overlapping-intervals", "Non-overlapping Intervals", "435", content.DifficultyMedium, []string{"intervals", "greedy"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 48},
	{"meeting-rooms-ii", "Meeting Rooms II", "253", content.DifficultyMedium, []string{"intervals", "heap"}, []content.ProblemSourceName{"blind75", "neetcode150", "grind75"}, 70},

	// Bit Manipulation
	{"single-number", "Single Number", "136", content.DifficultyEasy, []string{"bit-manipulation"}, []content.ProblemSourceName{"neetcode150"}, 44},
	{"number-of-1-bits", "Number of 1 Bits", "191", content.DifficultyEasy, []string{"bit-manipulation"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 40},
	{"counting-bits", "Counting Bits", "338", content.DifficultyEasy, []string{"bit-manipulation", "dynamic-programming"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 42},
	{"sum-of-two-integers", "Sum of Two Integers", "371", content.DifficultyMedium, []string{"bit-manipulation"}, []content.ProblemSourceName{"blind75", "neetcode150"}, 36},

	// Math & Geometry
	{"rotate-image", "Rotate Image", "48", content.DifficultyMedium, []string{"math"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 56},
	{"spiral-matrix", "Spiral Matrix", "54", content.DifficultyMedium, []string{"math"}, []content.ProblemSourceName{"neetcode150", "grind75"}, 58},
	{"set-matrix-zeroes", "Set Matrix Zeroes", "73", content.DifficultyMedium, []string{"math", "arrays-hashing"}, []content.ProblemSourceName{"neetcode150"}, 46},
}

// seedProblems upserts the canonical problem set, their pattern links, source
// memberships, and topic grouping. Idempotent via slug / composite keys.
func (s *Seeder) seedProblems(tx *gorm.DB, trackID uuid.UUID, patterns, dsaTopics map[string]uuid.UUID) error {
	for _, d := range canonicalProblems {
		// Topic grouping: the first pattern's slug-aligned topic.
		var topicID *uuid.UUID
		if len(d.PatternSlugs) > 0 {
			if tid, ok := dsaTopics[d.PatternSlugs[0]]; ok {
				topicID = &tid
			}
		}
		url := lcBase + d.Slug + "/"
		_, isPremium := premiumProblems[d.Slug]
		prob := content.Problem{
			TrackID:          trackID,
			TopicID:          topicID,
			Slug:             d.Slug,
			Title:            d.Title,
			Difficulty:       d.Difficulty,
			Platform:         content.ProblemPlatform("leetcode"),
			ExternalID:       ptr(d.LeetCodeID),
			URL:              ptr(url),
			EstimatedMinutes: estimatedMinutes(d.Difficulty),
			FrequencyScore:   d.Frequency,
			IsPremium:        isPremium,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoUpdates: clause.AssignmentColumns([]string{"track_id", "topic_id", "title", "difficulty", "platform", "external_id", "url", "estimated_minutes", "frequency_score", "is_premium", "updated_at"}),
		}).Create(&prob).Error; err != nil {
			return err
		}
		var got content.Problem
		if err := tx.Where("slug = ?", d.Slug).First(&got).Error; err != nil {
			return err
		}

		// Pattern links (dedup by (problem_id, pattern_id)).
		for _, ps := range d.PatternSlugs {
			pid, ok := patterns[ps]
			if !ok {
				continue
			}
			pp := content.ProblemPattern{ProblemID: got.ID, PatternID: pid}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "problem_id"}, {Name: "pattern_id"}},
				DoNothing: true,
			}).Create(&pp).Error; err != nil {
				return err
			}
		}

		// Source memberships (dedup by (problem_id, source)).
		for rank, src := range d.Sources {
			ps := content.ProblemSource{
				ProblemID:  got.ID,
				Source:     src,
				SourceRank: ptr(rank + 1),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "problem_id"}, {Name: "source"}},
				DoUpdates: clause.AssignmentColumns([]string{"source_rank", "updated_at"}),
			}).Create(&ps).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func estimatedMinutes(d content.Difficulty) int {
	switch d {
	case content.DifficultyEasy:
		return 20
	case content.DifficultyHard:
		return 50
	default:
		return 35
	}
}
