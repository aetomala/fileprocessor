package fileprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	StateStopped int32 = 0
	StateStarted int32 = 1
)

type Config struct {
	SourceDir      string
	OutputDir      string
	WorkerCount    int
	QueueSize      int
	ProcessTimeout time.Duration
}

type FileStats struct {
	Filename       string    `json:"filename"`
	WordCount      int       `json:"word_count"`
	LineCount      int       `json:"line_count"`
	CharacterCount int       `json:"character_count"`
	ProcessedAt    time.Time `json:"processed_at"`
}

type FileProcessor struct {
	config   Config
	jobQueue chan string // file name
	results  chan FileStats
	workers  sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	state    int32 // 0 stopped 1 running
	mu       sync.Mutex
}

func ConfigDefault() Config {
	return Config{
		WorkerCount:    3,
		QueueSize:      100,
		ProcessTimeout: 30 * time.Second,
	}
}

func (fp *FileProcessor) IsRunning() bool {
	state := atomic.LoadInt32(&fp.state)
	return state == StateStarted
}

func NewFileProcessor(config Config) (*FileProcessor, error) {
	if len(config.SourceDir) == 0 {
		return nil, fmt.Errorf("source directory is empty")
	}
	if len(config.OutputDir) == 0 {
		return nil, fmt.Errorf("output directory is empty")
	}
	if config.WorkerCount < 1 {
		config.WorkerCount = ConfigDefault().WorkerCount
	}
	if config.QueueSize < 1 {
		config.QueueSize = ConfigDefault().QueueSize
	}
	if config.ProcessTimeout < 1 {
		config.ProcessTimeout = ConfigDefault().ProcessTimeout
	}
	return &FileProcessor{
		config:   config,
		jobQueue: make(chan string, config.QueueSize),    // ADD THIS
		results:  make(chan FileStats, config.QueueSize), // ADD THIS
		state:    StateStopped,
	}, nil
}

func (fp *FileProcessor) Start(ctx context.Context) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	// Validate directories exist
	if _, err := os.Stat(fp.config.SourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", fp.config.SourceDir)
	}
	if _, err := os.Stat(fp.config.OutputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory does not exist: %s", fp.config.OutputDir)
	}

	// Check context before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check if already started
	if !atomic.CompareAndSwapInt32(&fp.state, StateStopped, StateStarted) {
		return fmt.Errorf("processor already started")
	}

	// Create cancellable context from the passed context
	fp.ctx, fp.cancel = context.WithCancel(ctx)

	// Initialize channels here if not done in constructor
	if fp.jobQueue == nil {
		fp.jobQueue = make(chan string, fp.config.QueueSize)
	}
	if fp.results == nil {
		fp.results = make(chan FileStats, fp.config.QueueSize)
	}

	// Start workers
	for i := 1; i <= fp.config.WorkerCount; i++ {
		fp.workers.Add(1)
		go fp.worker(i)
	}

	return nil
}

// Worker processes files from the job queue
func (fp *FileProcessor) worker(id int) {
	defer fp.workers.Done()

	for {
		select {
		case job, ok := <-fp.jobQueue:
			if !ok {
				return // Job queue closed, worker exits
			}

			// Process the file with timeout
			ctx, cancel := context.WithTimeout(fp.ctx, fp.config.ProcessTimeout)
			stats, err := fp.processFileWithContext(ctx, job)
			cancel()

			if err != nil {
				// send empty stats
				fp.results <- FileStats{Filename: filepath.Base(job)}
				continue
			}

			// Send result
			select {
			case fp.results <- stats:
				// Result sent successfully
			case <-fp.ctx.Done():
				// Context cancelled, exit worker
				return
			}

		case <-fp.ctx.Done():
			return // Context cancelled, exit worker
		}
	}
}

func (fp *FileProcessor) processFileWithContext(ctx context.Context, filePath string) (FileStats, error) {
	// Check context before processing
	select {
	case <-ctx.Done():
		return FileStats{}, ctx.Err()
	default:
	}

	return fp.processFile(filePath)
}

func (fp *FileProcessor) processFile(filePath string) (FileStats, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return FileStats{}, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	contentStr := string(content)

	return FileStats{
		Filename:       filepath.Base(filePath),
		WordCount:      len(strings.Fields(contentStr)),
		LineCount:      strings.Count(contentStr, "\n") + 1, // +1 for last line if no trailing newline
		CharacterCount: len(contentStr),
		ProcessedAt:    time.Now(),
	}, nil
}

func (fp *FileProcessor) ProcessFiles() error {

	if !fp.IsRunning() {
		return fmt.Errorf("processor not started")
	}

	// Get list of .txt files
	pattern := filepath.Join(fp.config.SourceDir, "*.txt")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to scan source directory: %w", err)
	}

	// Handle empty directory case
	if len(files) == 0 {
		return nil // No files to process
	}

	// Send jobs to workers in separate goroutine
	go func() {
		defer close(fp.jobQueue)

		for _, filename := range files {
			select {
			case fp.jobQueue <- filename:
				// Job sent successfully
			case <-fp.ctx.Done():
				// Context cancelled, stop sending jobs
				atomic.CompareAndSwapInt32(&fp.state, StateStarted, StateStopped)
				return
			}
		}
	}()

	// Collect results
	var processedCount int
	expectedCount := len(files)

	for processedCount < expectedCount {
		select {
		case stats, ok := <-fp.results:
			if !ok {
				// Results channel closed unexpectedly
				return fmt.Errorf("results channel closed unexpectedly")
			}

			// Write result to output file
			err := fp.writeResult(stats)
			if err != nil {
				return fmt.Errorf("failed to write result: %w", err)
			}

			processedCount++

		case <-fp.ctx.Done():
			atomic.CompareAndSwapInt32(&fp.state, StateStarted, StateStopped)
			return fp.ctx.Err()
		}
	}

	return nil
}

func (fp *FileProcessor) writeResult(stats FileStats) error {
	if stats.WordCount >= 0 {
		jsonData, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		// Use .json extension for result files
		outputFile := filepath.Join(fp.config.OutputDir, stats.Filename+".json")
		err = os.WriteFile(outputFile, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
	}

	return nil
}

func (fp *FileProcessor) Shutdown() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if !atomic.CompareAndSwapInt32(&fp.state, StateStarted, StateStopped) {
		// Already stopped
		return nil
	}

	// Cancel context to signal workers to stop
	if fp.cancel != nil {
		fp.cancel()
	}

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		fp.workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers finished cleanly
	case <-time.After(5 * time.Second):
		// Timeout waiting for workers
		return fmt.Errorf("timeout waiting for workers to shutdown")
	}

	return nil
}
