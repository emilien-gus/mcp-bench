package main

import (
	"encoding/csv"
	"fmt"
	"mcp-bench/internal/bench"
	"os"
	"strconv"
	"time"
)

type experiment struct {
	transport   string
	tool        string
	n           int
	mode        bench.Mode
	concurrency int
}

func main() {
	httpURL := "http://localhost:8080/mcp"
	serverBin := "./mcp-server"

	experiments := []experiment{
		// sequential — все payload
		{"http", "echo", 10000, bench.ModeCall, 1},
		{"http", "heavy", 10000, bench.ModeCall, 1},
		{"http", "ultra", 500, bench.ModeCall, 1},
		{"http", "superheavy", 50, bench.ModeCall, 1},
		{"stdio", "echo", 10000, bench.ModeCall, 1},
		{"stdio", "heavy", 10000, bench.ModeCall, 1},
		{"stdio", "ultra", 500, bench.ModeCall, 1},
		{"stdio", "superheavy", 50, bench.ModeCall, 1},

		// concurrent — echo и heavy
		{"http", "echo", 10000, bench.ModeCall, 2},
		{"http", "echo", 10000, bench.ModeCall, 4},
		{"http", "echo", 10000, bench.ModeCall, 8},
		{"http", "echo", 10000, bench.ModeCall, 16},
		{"http", "heavy", 10000, bench.ModeCall, 2},
		{"http", "heavy", 10000, bench.ModeCall, 4},
		{"http", "heavy", 10000, bench.ModeCall, 8},
		{"http", "heavy", 10000, bench.ModeCall, 16},
		{"stdio", "echo", 10000, bench.ModeCall, 2},
		{"stdio", "echo", 10000, bench.ModeCall, 4},
		{"stdio", "echo", 10000, bench.ModeCall, 8},
		{"stdio", "echo", 10000, bench.ModeCall, 16},
		{"stdio", "heavy", 10000, bench.ModeCall, 2},
		{"stdio", "heavy", 10000, bench.ModeCall, 4},
		{"stdio", "heavy", 10000, bench.ModeCall, 8},
		{"stdio", "heavy", 10000, bench.ModeCall, 16},

		// session
		{"http", "echo", 10000, bench.ModeSession, 1},
		{"http", "heavy", 10000, bench.ModeSession, 1},
		{"stdio", "echo", 10000, bench.ModeSession, 1},
		{"stdio", "heavy", 10000, bench.ModeSession, 1},
	}

	os.MkdirAll("results", 0755)

	callFile, _ := os.Create("results/call.csv")
	defer callFile.Close()
	callWriter := csv.NewWriter(callFile)
	callWriter.Write([]string{"transport", "tool", "n", "concurrency", "avg_ms", "p95_ms", "p99_ms", "max_ms", "errors"})

	sessionFile, _ := os.Create("results/session.csv")
	defer sessionFile.Close()
	sessionWriter := csv.NewWriter(sessionFile)
	sessionWriter.Write([]string{"transport", "tool", "n", "total_ms", "errors"})

	for _, exp := range experiments {
		fmt.Printf("Running: transport=%s tool=%s n=%d mode=%s concurrency=%d ...\n",
			exp.transport, exp.tool, exp.n, exp.mode, exp.concurrency)

		var result *bench.Result
		var err error

		if exp.transport == "http" {
			result, err = bench.Run(bench.Config{
				Tool:        exp.tool,
				N:           exp.n,
				ServerURL:   httpURL,
				Mode:        exp.mode,
				Concurrency: exp.concurrency,
			})
		} else {
			result, err = bench.RunStdioBench(serverBin, exp.tool, exp.n, exp.mode, exp.concurrency)
		}

		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		label := fmt.Sprintf("%s/%s/n=%d/c=%d/%s",
			exp.transport, exp.tool, exp.n, exp.concurrency, exp.mode)

		if exp.mode == bench.ModeCall {
			stats := result.Stats()
			stats.Print(label)
			callWriter.Write([]string{
				exp.transport, exp.tool,
				strconv.Itoa(exp.n),
				strconv.Itoa(exp.concurrency),
				fmtMs(stats.Avg), fmtMs(stats.P95),
				fmtMs(stats.P99), fmtMs(stats.Max),
				strconv.Itoa(stats.Errors),
			})
			callWriter.Flush()
		} else {
			total := result.Total()
			fmt.Printf("\n=== %s ===\nTotal: %v\n", label, total)
			sessionWriter.Write([]string{
				exp.transport, exp.tool,
				strconv.Itoa(exp.n),
				fmtMs(total),
				strconv.Itoa(result.Errors),
			})
			sessionWriter.Flush()
		}
	}

	fmt.Println("\nDone! Results saved to results/")
}

func fmtMs(d time.Duration) string {
	return fmt.Sprintf("%.4f", float64(d.Nanoseconds())/1_000_000.0)
}
