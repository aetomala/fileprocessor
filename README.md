# File Processing Worker Pool Project

## Project Overview
Build a concurrent file processor that uses a worker pool pattern to process text files. The system should read files from a source directory, process them (word counting), and write results to an output directory.

## Core Requirements

### Worker Pool Configuration
- Configurable number of workers (default: 3)
- Configurable queue buffer size (default: 100)
- Configurable processing timeout per file (default: 30s)

### File Processing Logic
- Input: Text files from a source directory
- Processing: Count total words, lines, and characters in each file
- Output: JSON results file with statistics per processed file

### Expected Behavior
1. **Initialization**: Create worker pool with specified number of workers
2. **Job Discovery**: Scan source directory for .txt files
3. **Job Distribution**: Distribute files to available workers via buffered channel
4. **Processing**: Each worker processes files concurrently and generates statistics
5. **Result Collection**: Collect results and write to output directory
6. **Graceful Shutdown**: Complete in-flight work when context is cancelled

### Data Structures

```go
type Config struct {
    SourceDir      string
    OutputDir      string
    WorkerCount    int
    QueueSize      int
    ProcessTimeout time.Duration
}

type FileStats struct {
    Filename      string    `json:"filename"`
    WordCount     int       `json:"word_count"`
    LineCount     int       `json:"line_count"`
    CharacterCount int      `json:"character_count"`
    ProcessedAt   time.Time `json:"processed_at"`
}

type FileProcessor struct {
    config    Config
    jobQueue  chan string
    results   chan FileStats
    workers   sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
    running   atomic.Bool
}
```

### Core Methods to Implement

```go
func NewFileProcessor(config Config) (*FileProcessor, error)
func (fp *FileProcessor) Start(ctx context.Context) error
func (fp *FileProcessor) ProcessFiles() error
func (fp *FileProcessor) IsRunning() bool
func (fp *FileProcessor) Shutdown() error
func (fp *FileProcessor) processFile(filename string) (FileStats, error)
func (fp *FileProcessor) worker(id int)
func (fp *FileProcessor) collectResults()
```

### Success Criteria
- Process multiple files concurrently
- Handle file I/O errors gracefully
- Respect context cancellation
- Generate accurate word/line/character counts
- Write results in valid JSON format
- No goroutine leaks or race conditions

### Sample Input/Output

**Input file (sample.txt):**
```
Hello world!
This is a test file.
It has multiple lines.
```

**Output file (results.json):**
```json
{
  "filename": "sample.txt",
  "word_count": 10,
  "line_count": 3,
  "character_count": 50,
  "processed_at": "2025-01-15T10:30:00Z"
}
```

## Implementation Notes
- Use `strings.Fields()` for word counting
- Use `strings.Count()` for line counting
- Handle edge cases: empty files, permission errors, invalid UTF-8
- Ensure proper cleanup of goroutines and channels
- Use atomic operations for thread-safe state management