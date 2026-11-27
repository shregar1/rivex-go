package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func main() {
	var (
		benchtime  = flag.Duration("benchtime", 5*time.Second, "Duration to run each benchmark")
		cpu        = flag.Int("cpu", runtime.NumCPU(), "Number of CPUs to use")
		mem        = flag.Bool("mem", true, "Include memory allocation stats")
		output     = flag.String("o", "table", "Output format: table, csv, json")
		benchRegex = flag.String("filter", "BenchmarkJSON|BenchmarkPlainText|BenchmarkStreaming", "Benchmark filter regex")
	)
	flag.Parse()
	if err := runBenchmarks(*benchtime, *cpu, *mem, *benchRegex, *output); err != nil {
		fmt.Fprintf(os.Stderr, "bench failed: %v\n", err)
		os.Exit(1)
	}
}

func runBenchmarks(benchtime time.Duration, cpu int, mem bool, filter, output string) error {
	testsDir, _ := filepath.Abs(".")
	if _, err := os.Stat(testsDir); err != nil {
		return fmt.Errorf("tests dir not found: %w", err)
	}
	binPath := filepath.Join(os.TempDir(), "rivex-bench-test")
	fmt.Printf("Rivex Benchmark Suite\n")
	fmt.Printf("==========================================\n")
	fmt.Printf("Platform:  %s/%s, %d CPUs\n", runtime.GOOS, runtime.GOARCH, cpu)
	fmt.Printf("Duration:  %v per benchmark\n", benchtime)
	fmt.Printf("Memory:    %v\n\n", mem)
	fmt.Println("Building benchmark binary...")
	build := exec.Command("go", "test", "-c", "-o", binPath, "./tests/...")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	args := []string{
		"-test.bench=" + filter,
		"-test.benchtime=" + benchtime.String(),
		"-test.cpu=" + fmt.Sprintf("%d", cpu),
		"-test.run=^$",
	}
	if mem {
		args = append(args, "-test.benchmem")
	}
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
