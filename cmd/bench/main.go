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
	transport string
	tool      string
	n         int
	mode      bench.Mode
}

func main() {
	httpURL := "http://localhost:8080/mcp"
	serverBin := "./mcp-server.exe"

	experiments := []experiment{
		{"http", "echo", 1000, bench.ModeCall},
		{"http", "echo", 10000, bench.ModeCall},
		{"http", "heavy", 1000, bench.ModeCall},
		{"http", "heavy", 10000, bench.ModeCall},
		{"stdio", "echo", 1000, bench.ModeCall},
		{"stdio", "echo", 10000, bench.ModeCall},
		{"stdio", "heavy", 1000, bench.ModeCall},
		{"stdio", "heavy", 10000, bench.ModeCall},
		{"http", "echo", 1000, bench.ModeSession},
		{"http", "echo", 10000, bench.ModeSession},
		{"http", "heavy", 1000, bench.ModeSession},
		{"http", "heavy", 10000, bench.ModeSession},
		{"stdio", "echo", 1000, bench.ModeSession},
		{"stdio", "echo", 10000, bench.ModeSession},
		{"stdio", "heavy", 1000, bench.ModeSession},
		{"stdio", "heavy", 10000, bench.ModeSession},
	}

	os.MkdirAll("results", 0755)

	// два отдельных CSV файла
	callFile, _ := os.Create("results/call.csv")
	defer callFile.Close()
	callWriter := csv.NewWriter(callFile)
	callWriter.Write([]string{"transport", "tool", "n", "avg_ms", "p95_ms", "p99_ms", "max_ms", "errors"})

	sessionFile, _ := os.Create("results/session.csv")
	defer sessionFile.Close()
	sessionWriter := csv.NewWriter(sessionFile)
	sessionWriter.Write([]string{"transport", "tool", "n", "total_ms", "errors"})

	for _, exp := range experiments {
		fmt.Printf("Running: transport=%s tool=%s n=%d mode=%s ...\n",
			exp.transport, exp.tool, exp.n, exp.mode)

		var result *bench.Result
		var err error

		if exp.transport == "http" {
			cfg := bench.Config{
				Tool: exp.tool, N: exp.n,
				ServerURL: httpURL, Mode: exp.mode,
			}
			result, err = bench.Run(cfg)
		} else {
			result, err = bench.RunStdioBench(serverBin, exp.tool, exp.n, exp.mode)
		}

		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		label := fmt.Sprintf("%s / %s / n=%d / %s", exp.transport, exp.tool, exp.n, exp.mode)

		if exp.mode == bench.ModeCall {
			stats := result.Stats()
			stats.Print(label)
			callWriter.Write([]string{
				exp.transport, exp.tool,
				strconv.Itoa(exp.n),
				fmtMs(stats.Avg),
				fmtMs(stats.P95),
				fmtMs(stats.P99),
				fmtMs(stats.Max),
				strconv.Itoa(stats.Errors),
			})
			callWriter.Flush()
		} else {
			total := result.Total()
			fmt.Printf("\n=== %s ===\n", label)
			fmt.Printf("Total: %v\n", total)
			sessionWriter.Write([]string{
				exp.transport, exp.tool,
				strconv.Itoa(exp.n),
				fmtMs(total),
				strconv.Itoa(result.Errors),
			})
			sessionWriter.Flush()
		}
	}

	fmt.Println("\nResults saved to results/call.csv and results/session.csv")
}

func fmtMs(d time.Duration) string {
	return fmt.Sprintf("%.4f", float64(d.Nanoseconds())/1_000_000.0)
}
