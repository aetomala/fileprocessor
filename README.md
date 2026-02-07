# fileprocessor

![Tests](https://github.com/aetomala/fileprocessor/actions/workflows/test.yml/badge.svg?branch=main)
![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

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
```go
type FileProcessor struct {
    config   Config
    jobQueue chan string        // Buffered channel for work distribution
    results  chan FileStats     // Result collection channel
    workers  sync.WaitGroup     // Worker lifecycle management
    ctx      context.Context    // Cancellation propagation
    cancel   context.CancelFunc
    state    int32              // Atomic state (Stopped=0, Started=1)
    mu       sync.Mutex         // Protects Start/Shutdown operations
}
```

**Key patterns implemented**:
- **Worker Pool**: Bounded parallelism with configurable worker count
- **Context Propagation**: Cancellation signals flow through all workers
- **Graceful Shutdown**: WaitGroup ensures in-flight work completes
- **Atomic State**: Lock-free state checks with CompareAndSwap
- **Channel Coordination**: Buffered channels for backpressure handling
- **Timeout Management**: Per-file processing timeouts with context
- **Error Recovery**: Individual file failures don't crash the pipeline

## Testing Approach

Built incrementally using TDD with comprehensive test coverage (30+ test cases):

**Phase 1: Core Functionality**
- Constructor validation and configuration defaults
- Lifecycle management (Start/Stop/IsRunning)
- Single and multi-file processing
- File filtering (.txt only) and statistics generation

**Phase 2: Concurrency & Performance**
- Worker pool validation (1-10 workers)
- Large file set processing (50+ files)
- Mixed file size handling (10B to 1MB)
- Queue overflow and backpressure handling

**Phase 3: Error Handling**
- File permission errors (unreadable files)
- Output directory write failures
- Processing timeouts (nanosecond precision)
- Corrupted UTF-8 file handling

**Phase 4: Context Management**
- Context cancellation during processing
- In-flight work completion
- Timeout enforcement and propagation

**Phase 5: Shutdown & Resource Cleanup**
- Graceful shutdown (idle and active states)
- Concurrent shutdown calls
- Worker timeout handling
- Goroutine leak detection (runtime validation)
- Rapid start/shutdown cycles

**Phase 6: Edge Cases**
- Shutdown during initialization
- State consistency under concurrent operations
- Channel closure verification
- Empty directory handling

All tests run with race detection (`-race` flag) to ensure memory safety.

## Test Coverage Highlights

**30+ test cases** organized into comprehensive test suites:

**Basic Functionality** (15 tests)
- Constructor validation with required/optional fields
- Start/Stop lifecycle with double-start prevention
- Single and multiple file processing
- File type filtering and accurate statistics
- Empty directory handling

**Advanced Features** (6 tests)
- Large-scale processing (50 files, <5s completion)
- Mixed file sizes (10B to 1MB)
- Corrupted UTF-8 handling without crashes
- Dynamic worker scaling (1, 2, 5, 10 workers)

**Shutdown Testing** (9 tests)
- Graceful shutdown (idle and active states)
- In-flight work completion guarantees
- Multiple/concurrent shutdown calls
- Goroutine leak detection (±2 tolerance)
- Rapid start/shutdown cycles (5 iterations)
- Shutdown during initialization edge case
- State consistency under concurrent operations

## Running the Code
```bash
# Run all tests with race detection
ginkgo -race cmd/fileprocessor

# Run tests with verbose output
ginkgo -v cmd/fileprocessor

# Run specific test suite
ginkgo -focus "Shutdown" cmd/fileprocessor

# Standard Go test runner
go test -race ./cmd/fileprocessor/...

# Build CLI (when implemented)
go build -o fileprocessor ./cmd/fileprocessor
```

## Implementation Status

✅ **Core Implementation Complete** - Production-ready with comprehensive test coverage

**Completed Features**:
- ✅ Core data structures and interfaces
- ✅ Configuration management with sensible defaults
- ✅ Worker pool with configurable concurrency (1-10+ workers)
- ✅ File processing logic (word/line/character counting)
- ✅ Result collection pipeline with JSON output
- ✅ Context-based cancellation and timeout management
- ✅ Graceful shutdown with in-flight work completion
- ✅ Comprehensive error handling and recovery
- ✅ Test coverage (30+ test cases) with race detection
- ✅ Resource leak validation (goroutine tracking)

**Validated Scenarios**:
- Concurrent processing under load (50+ files)
- Context cancellation during active processing
- Timeout enforcement (nanosecond to second precision)
- Corrupted file handling (invalid UTF-8)
- File permission errors and output failures
- Shutdown edge cases (concurrent calls, initialization timing)

**Future Enhancements**:
- CLI argument parsing and main() implementation
- Performance benchmarking suite
- Memory profiling and optimization
- Prometheus metrics export
- Distributed tracing integration

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