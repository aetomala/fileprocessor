package fileprocessor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	StateStopped = 0
	StateStarted = 1
)

type Config struct {
	SourceDir      string
	OutputDir      string
	WorkerCount    int           `default:"3"`
	QueueSize      int           `default:"100"`
	ProcessTimeout time.Duration `default:"30 * time.Second"`
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
	jobQueue chan string
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

func (fp *FileProcessor) Shutdown() error {
	return nil
}

func (fp *FileProcessor) IsRunning() bool {
	return atomic.LoadInt32(&fp.state) == StateStarted
}

func NewFileProcessor(config Config) (*FileProcessor, error) {
	if len(config.SourceDir) == 0 || len(config.OutputDir) == 0 {
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
		config: config,
	}, nil
}

// Start starts worker pool
func (fp *FileProcessor) Start(ctx context.Context) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	_, err := os.Stat(fp.config.SourceDir)
	if err != nil {
		if os.ErrNotExist == err {
			return fmt.Errorf("source does not exist %s", fp.config.SourceDir)
		}
		return fmt.Errorf("error validating source directory: %s", fp.config.SourceDir)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if !atomic.CompareAndSwapInt32(&fp.state, StateStopped, StateStarted) {
			return fmt.Errorf("already started")
		}

		fp.ctx, fp.cancel = context.WithTimeout(ctx, 10*time.Second)

		// launch the go routines
		for i := 1; i <= fp.config.WorkerCount; i++ {
			fp.workers.Add(1)
			go worker(fp.ctx, fp.jobQueue, fp.results, &fp.workers)
		}
	}

	return nil
}

// Worker is responsible for executing the work
func worker(ctx context.Context, jobs <-chan string, results chan<- FileStats, wg *sync.WaitGroup) {
	defer wg.Done()
}
