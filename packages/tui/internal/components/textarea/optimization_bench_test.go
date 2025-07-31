package textarea

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkHashingPerformance compares SHA256 vs FNV-1a hashing performance
func BenchmarkHashingPerformance(b *testing.B) {
	testData := []string{
		"short",
		"medium length string for testing",
		strings.Repeat("long string for performance testing ", 100),
	}

	b.Run("FNV-1a", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, data := range testData {
				h := HString(data)
				_ = h.Hash() // Uses FNV-1a (current version)
			}
		}
	})

	b.Run("SHA256", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, data := range testData {
				h := HString(data)
				_ = h.HashWithVersion(HashVersionV1) // Uses SHA256 (legacy)
			}
		}
	})
}

// BenchmarkMemoryOptimizations compares old vs new memory allocation strategies
func BenchmarkMemoryOptimizations(b *testing.B) {
	// Create test data with mixed content
	testItems := make([]any, 1000)
	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			// 10% attachments
			testItems[i] = &Attachment{Display: fmt.Sprintf("attachment_%d", i)}
		} else {
			// 90% runes
			testItems[i] = rune('A' + (i % 26))
		}
	}

	b.Run("interfacesToRunes_Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := interfacesToRunes(testItems)
			_ = result
		}
	})

	b.Run("interfacesToString_Optimized", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := interfacesToString(testItems)
			_ = result
		}
	})

	// Benchmark the old naive approach for comparison
	b.Run("interfacesToRunes_Naive", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Naive approach: no preallocation
			var result []rune
			for _, item := range testItems {
				switch val := item.(type) {
				case rune:
					result = append(result, val)
				case *Attachment:
					result = append(result, []rune(val.Display)...)
				}
			}
			_ = result
		}
	})

	b.Run("interfacesToString_Naive", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Naive approach: no capacity estimation
			var s strings.Builder
			for _, item := range testItems {
				switch val := item.(type) {
				case rune:
					s.WriteRune(val)
				case *Attachment:
					s.WriteString(val.Display)
				}
			}
			_ = s.String()
		}
	})
}

// BenchmarkCachePerformance tests the performance of the memoization cache
func BenchmarkCachePerformance(b *testing.B) {
	cache := NewMemoCache[HString, string](1000)
	migrationCache := NewMigrationAwareCache[HString, string](1000, 0) // No migration period

	testKeys := make([]HString, 100)
	for i := 0; i < 100; i++ {
		testKeys[i] = HString(fmt.Sprintf("test_key_%d", i))
	}

	// Pre-populate cache
	for i, key := range testKeys {
		value := fmt.Sprintf("value_%d", i)
		cache.Set(key, value)
		migrationCache.Set(key, value)
	}

	b.Run("StandardCache_Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := testKeys[i%len(testKeys)]
			_, _ = cache.Get(key)
		}
	})

	b.Run("MigrationAwareCache_Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := testKeys[i%len(testKeys)]
			_, _ = migrationCache.Get(key)
		}
	})

	b.Run("StandardCache_Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := HString(fmt.Sprintf("bench_key_%d", i))
			value := fmt.Sprintf("bench_value_%d", i)
			cache.Set(key, value)
		}
	})

	b.Run("MigrationAwareCache_Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := HString(fmt.Sprintf("bench_key_%d", i))
			value := fmt.Sprintf("bench_value_%d", i)
			migrationCache.Set(key, value)
		}
	})
}

// BenchmarkCapacityEstimation tests the performance of our capacity estimation algorithms
func BenchmarkCapacityEstimation(b *testing.B) {
	// Create test data of various sizes
	smallItems := make([]any, 10)
	mediumItems := make([]any, 100)
	largeItems := make([]any, 1000)

	for i := range smallItems {
		smallItems[i] = rune('A' + (i % 26))
	}
	for i := range mediumItems {
		if i%5 == 0 {
			mediumItems[i] = &Attachment{Display: "attachment"}
		} else {
			mediumItems[i] = rune('A' + (i % 26))
		}
	}
	for i := range largeItems {
		if i%10 == 0 {
			largeItems[i] = &Attachment{Display: fmt.Sprintf("attachment_%d", i)}
		} else {
			largeItems[i] = rune('A' + (i % 26))
		}
	}

	b.Run("EstimateRuneCapacity_Small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = estimateRuneCapacity(smallItems)
		}
	})

	b.Run("EstimateRuneCapacity_Medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = estimateRuneCapacity(mediumItems)
		}
	})

	b.Run("EstimateRuneCapacity_Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = estimateRuneCapacity(largeItems)
		}
	})

	b.Run("EstimateStringCapacity_Small", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = estimateStringCapacity(smallItems)
		}
	})

	b.Run("EstimateStringCapacity_Medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = estimateStringCapacity(mediumItems)
		}
	})

	b.Run("EstimateStringCapacity_Large", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = estimateStringCapacity(largeItems)
		}
	})
}

// BenchmarkObjectPooling tests the performance impact of object pooling
func BenchmarkObjectPooling(b *testing.B) {
	testData := make([]rune, 100)
	for i := range testData {
		testData[i] = rune('A' + (i % 26))
	}

	b.Run("WithObjectPooling", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := runesToInterfaces(testData)
			_ = result
			// In a real scenario, we'd call putAnySlice(result) when done
		}
	})

	b.Run("WithoutObjectPooling", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Direct allocation without pooling
			result := make([]any, len(testData))
			for j, r := range testData {
				result[j] = r
			}
			_ = result
		}
	})
}
