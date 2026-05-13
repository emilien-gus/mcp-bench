package bench

import (
	"fmt"
	"sort"
	"time"
)

type Result struct {
	Latencies []time.Duration
	Errors    int
}

func (r *Result) Add(d time.Duration) {
	r.Latencies = append(r.Latencies, d)
}

func (r *Result) Stats() Stats {
	if len(r.Latencies) == 0 {
		return Stats{}
	}

	sorted := make([]time.Duration, len(r.Latencies))
	copy(sorted, r.Latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	total := time.Duration(0)
	for _, l := range sorted {
		total += l
	}

	n := len(sorted)
	return Stats{
		Count:  n,
		Avg:    total / time.Duration(n),
		Min:    sorted[0],
		P95:    percentile(sorted, 0.95),
		P99:    percentile(sorted, 0.99),
		Max:    sorted[n-1],
		Errors: r.Errors,
	}
}

type Stats struct {
	Count  int
	Avg    time.Duration
	Min    time.Duration
	P95    time.Duration
	P99    time.Duration
	Max    time.Duration
	Errors int
}

func (s Stats) Print(label string) {
	fmt.Printf("\n=== %s ===\n", label)
	fmt.Printf("Requests : %d (errors: %d)\n", s.Count, s.Errors)
	fmt.Printf("Avg      : %v\n", s.Avg)
	fmt.Printf("Min      : %v\n", s.Min)
	fmt.Printf("P95      : %v\n", s.P95)
	fmt.Printf("P99      : %v\n", s.P99)
	fmt.Printf("Max      : %v\n", s.Max)
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func (r *Result) Total() time.Duration {
	total := time.Duration(0)
	for _, l := range r.Latencies {
		total += l
	}
	return total
}
