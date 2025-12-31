#!/bin/bash
# Comprehensive benchmark suite for slug architecture improvements
# Date: 2025-12-30

set -e

echo "======================================"
echo "Slug Architecture Performance Testing"
echo "======================================"
echo ""

# Create output directory
mkdir -p ../benchmarks

cd /home/jwwelbor/projects/shark-task-manager

echo "[1/4] Running PathResolver benchmarks..."
go test -bench=BenchmarkPathResolver -benchmem -benchtime=5s -run=^$ \
    github.com/jwwelbor/shark-task-manager/internal/pathresolver 2>&1 | \
    tee dev-artifacts/2025-12-30-slug-architecture-performance/benchmarks/pathresolver-detailed.txt

echo ""
echo "[2/4] Running Repository GetByKey benchmarks..."
go test -bench=BenchmarkEpic -benchmem -benchtime=5s -run=^$ \
    github.com/jwwelbor/shark-task-manager/internal/repository 2>&1 | \
    tee dev-artifacts/2025-12-30-slug-architecture-performance/benchmarks/repository-detailed.txt

echo ""
echo "[3/4] Running Status Dashboard benchmarks..."
go test -bench=BenchmarkGetDashboard -benchmem -benchtime=3s -run=^$ \
    github.com/jwwelbor/shark-task-manager/internal/status 2>&1 | \
    tee dev-artifacts/2025-12-30-slug-architecture-performance/benchmarks/status-detailed.txt

echo ""
echo "[4/4] Running Query Plan analysis..."
go test -v -run=TestQueryPlanAnalysis \
    github.com/jwwelbor/shark-task-manager/internal/repository 2>&1 | \
    tee dev-artifacts/2025-12-30-slug-architecture-performance/benchmarks/query-plan-analysis.txt

echo ""
echo "======================================"
echo "Benchmark suite complete!"
echo "Results saved to dev-artifacts/2025-12-30-slug-architecture-performance/benchmarks/"
echo "======================================"
