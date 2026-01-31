# fileprocessor

**Concurrent file processing patterns in Go - exploring worker pools, graceful shutdown, and backpressure handling**

## Purpose

Part of my ongoing platform engineering skill maintenance - this repo explores Go concurrency patterns through a practical file processing implementation. Built incrementally with TDD using Ginkgo/Gomega, each feature targets production patterns I've used in distributed systems work.

**Why file processing?** It's a domain-agnostic way to explore worker pool patterns, context propagation, and resource management - fundamentals that appear in log pipelines, ETL systems, and distributed task processing across platform infrastructure.

## Current Implementation

Building a concurrent file processor with:
- **Worker pool pattern** with configurable concurrency (N workers)
- **Buffered job queue** with backpressure handling
- **Context-based cancellation** and timeout management
- **Graceful shutdown** ensuring work completion
- **Result collection** with concurrent-safe aggregation

Processing pipeline: Scan directory → Queue files → Distribute to workers → Process (word/line/char counting) → Collect results → Write JSON output

## Architecture
```
type FileProcessor struct {
    config    Config
    jobQueue  chan string        // Buffered channel for work distribution
    results   chan FileStats     // Result collection channel
    workers   sync.WaitGroup     // Worker lifecycle management
    ctx       context.Context    // Cancellation propagation
    cancel    context.CancelFunc
    running   atomic.Bool        // Thread-safe state
}
```

**Key patterns explored**:
- Bounded parallelism with semaphore pattern
- Context deadline propagation through pipeline
- Coordinated shutdown using WaitGroup
- Atomic operations for concurrent state

## Testing Approach

Following TDD with progressive complexity:
1. **Phase 1**: Single worker, single file (happy path)
2. **Phase 2**: Multiple workers, concurrent processing
3. **Phase 3**: Error handling, timeout scenarios
4. **Phase 4**: Graceful shutdown, resource cleanup
5. **Phase 5**: Edge cases (empty files, permissions, invalid UTF-8)

Using Ginkgo/Gomega for BDD-style tests with explicit concurrency validation.

## Running the Code
```bash
# Run tests
go test -race ./...

# Run with Ginkgo
ginkgo -v ./...

# Build CLI
go build -o fileprocessor ./cmd/fileprocessor

# Process files
./fileprocessor --source ./input --output ./output --workers 5
```

## Implementation Status

🚧 **Active Development** - Building incrementally with TDD

**Completed**:
- Core data structures and interfaces
- Configuration management
- Worker pool scaffolding

**In Progress**:
- File processing logic
- Result collection pipeline
- Test coverage for concurrent scenarios

**Planned**:
- Performance benchmarking
- Memory profiling
- CLI argument parsing

## Why This Pattern Matters

File processing with worker pools is a microcosm of production distributed systems:
- **Resource-bounded parallelism** (like Kubernetes pod limits)
- **Backpressure handling** (like message queue consumers)
- **Graceful degradation** (like rolling deployments)
- **Observable failures** (like distributed tracing)

These patterns scale from single-machine file processing to multi-region data pipelines.

## Background

I'm a Senior Platform Engineer with 28 years of experience building distributed systems. This repo represents deliberate practice - keeping Go concurrency fundamentals sharp between infrastructure projects. Not everything needs to be production code; sometimes the process of building is the point.

## Development Approach

Built using AI-assisted pair programming (Claude) to explore modern workflows while maintaining rigorous engineering standards. All code follows TDD with comprehensive test coverage.

## License

MIT