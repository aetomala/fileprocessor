package fileprocessor_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"fileprocessor/cmd/fileprocessor"
)

var _ = Describe("FileProcessor", func() {
	var (
		processor *fileprocessor.FileProcessor
		ctx       context.Context
		cancel    context.CancelFunc
		tempDir   string
		sourceDir string
		outputDir string
		config    fileprocessor.Config
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)

		// Create temporary directories for testing
		var err error
		tempDir, err = os.MkdirTemp("", "fileprocessor_test")
		Expect(err).NotTo(HaveOccurred())

		sourceDir = filepath.Join(tempDir, "source")
		outputDir = filepath.Join(tempDir, "output")

		err = os.MkdirAll(sourceDir, 0755)
		Expect(err).NotTo(HaveOccurred())
		err = os.MkdirAll(outputDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		config = fileprocessor.Config{
			SourceDir:      sourceDir,
			OutputDir:      outputDir,
			WorkerCount:    2,
			QueueSize:      10,
			ProcessTimeout: 5 * time.Second,
		}
	})

	AfterEach(func() {
		if cancel != nil {
			cancel()
		}
		if processor != nil && processor.IsRunning() {
			err := processor.Shutdown()
			Expect(err).NotTo(HaveOccurred())
		}
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	// === PHASE 1: Basic Construction ===
	Describe("Constructor", func() {
		It("should create a new FileProcessor with valid configuration", func() {
			var err error
			processor, err = fileprocessor.NewFileProcessor(config)

			Expect(err).NotTo(HaveOccurred())
			Expect(processor).NotTo(BeNil())
			Expect(processor.IsRunning()).To(BeFalse())
		})

		It("should validate required configuration fields", func() {
			invalidConfigs := []fileprocessor.Config{
				{SourceDir: "", OutputDir: outputDir, WorkerCount: 1, QueueSize: 10, ProcessTimeout: time.Second},
				{SourceDir: sourceDir, OutputDir: "", WorkerCount: 1, QueueSize: 10, ProcessTimeout: time.Second},
			}

			for _, cfg := range invalidConfigs {
				invalidProcessor, err := fileprocessor.NewFileProcessor(cfg)
				Expect(err).To(HaveOccurred())
				Expect(invalidProcessor).To(BeNil())
			}
		})

		It("should apply default values for optional configuration", func() {
			minimalConfig := fileprocessor.Config{
				SourceDir: sourceDir,
				OutputDir: outputDir,
				// WorkerCount, QueueSize, ProcessTimeout left as zero values
			}

			var err error
			processor, err = fileprocessor.NewFileProcessor(minimalConfig)

			Expect(err).NotTo(HaveOccurred())
			Expect(processor).NotTo(BeNil())

			// Test that defaults were applied by checking behavior
			err = processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Should work with default values
			testFile := filepath.Join(sourceDir, "default_test.txt")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = processor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle non-existent directories gracefully", func() {
			invalidDirConfig := fileprocessor.Config{
				SourceDir:      "/non/existent/source",
				OutputDir:      "/non/existent/output",
				WorkerCount:    1,
				QueueSize:      10,
				ProcessTimeout: time.Second,
			}

			// Constructor should succeed, validation happens at Start()
			var err error
			processor, err = fileprocessor.NewFileProcessor(invalidDirConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(processor).NotTo(BeNil())
		})
	})

	// === PHASE 2: Core Initialization ===
	Describe("Start", func() {
		BeforeEach(func() {
			var err error
			processor, err = fileprocessor.NewFileProcessor(config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should start successfully with valid configuration", func() {
			err := processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(processor.IsRunning()).To(BeTrue())
		})

		It("should validate directories exist at startup", func() {
			invalidConfig := fileprocessor.Config{
				SourceDir:      "/non/existent/source",
				OutputDir:      outputDir,
				WorkerCount:    1,
				QueueSize:      10,
				ProcessTimeout: time.Second,
			}

			invalidProcessor, err := fileprocessor.NewFileProcessor(invalidConfig)
			Expect(err).NotTo(HaveOccurred())

			err = invalidProcessor.Start(ctx)
			Expect(err).To(HaveOccurred())
			Expect(invalidProcessor.IsRunning()).To(BeFalse())
		})

		It("should prevent double start", func() {
			err := processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Second start should return error
			err = processor.Start(ctx)
			Expect(err).To(HaveOccurred())
			Expect(processor.IsRunning()).To(BeTrue())
		})

		It("should handle context cancellation during startup", func() {
			cancelledCtx, cancelFunc := context.WithCancel(ctx)
			cancelFunc() // Cancel immediately

			err := processor.Start(cancelledCtx)
			Expect(err).To(HaveOccurred())
			Expect(processor.IsRunning()).To(BeFalse())
		})

		It("should maintain IsRunning state correctly", func() {
			Expect(processor.IsRunning()).To(BeFalse())

			err := processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(processor.IsRunning()).To(BeTrue())
		})
	})

	// === PHASE 3: Basic Functionality ===
	Describe("Core File Processing", func() {
		BeforeEach(func() {
			var err error
			processor, err = fileprocessor.NewFileProcessor(config)
			Expect(err).ToNot(HaveOccurred())

			err = processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process a single text file successfully", func() {
			// Create test file
			testContent := "Hello world!\nThis is a test file.\nIt has multiple lines."
			testFile := filepath.Join(sourceDir, "test.txt")
			err := os.WriteFile(testFile, []byte(testContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = processor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())

			// Check result file was created
			resultFiles, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resultFiles)).To(Equal(1))
		})

		It("should handle empty source directory gracefully", func() {
			err := processor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())

			//No result files should be created
			resultFiles, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resultFiles)).To(Equal(0))
		})

		It("should process only .txt files", func() {
			//Create variaous file types
			files := map[string]string{
				"test.txt":  "Text content",
				"test.log":  "Log content",
				"test.json": `"{"key": "value}"`,
				"readme.":   "No extension",
			}
			for filename, content := range files {
				filePath := filepath.Join(sourceDir, filename)
				err := os.WriteFile(filePath, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())
			}

			err := processor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())

			// Only one result file should be create (for .txt file)
			resultFiles, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resultFiles)).To(Equal(1))
		})

		It("should generate accurate file statistics", func() {
			testContent := "Hello world!\nThis is a test.\nThree lines total."
			testFile := filepath.Join(sourceDir, "stats_test.txt")
			err := os.WriteFile(testFile, []byte(testContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = processor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())

			// Verify statistics in resul file
			resultFiles, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resultFiles)).To(Equal(1))

			// Read and verify the JSON content
			jsonData, err := os.ReadFile(resultFiles[0])
			Expect(err).NotTo(HaveOccurred())

			var stats fileprocessor.FileStats
			err = json.Unmarshal(jsonData, &stats)
			Expect(err).NotTo(HaveOccurred())

			Expect(stats.WordCount).To(Equal(9))
			Expect(stats.LineCount).To(Equal(3))
			Expect(stats.CharacterCount).To(Equal(len(testContent)))
		})
	})

	// == PHASE 4: Error Handling ===
	Describe("Error Handling", func() {
		BeforeEach(func() {
			var err error
			processor, err = fileprocessor.NewFileProcessor(config)
			Expect(err).NotTo(HaveOccurred())

			err = processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle undreadable files gracefully", func() {
			// Create file and remove read permissions
			testFile := filepath.Join(sourceDir, "unreadable.txt")
			err := os.WriteFile(testFile, []byte("content"), 0000) //no permissions
			Expect(err).NotTo(HaveOccurred())

			err = processor.ProcessFiles()
			//Should not fail the entire processing operation
			Expect(err).NotTo(HaveOccurred())
		})

		It("should habdle output directory write errors", func() {
			//Make output directory read-only
			err := os.Chmod(outputDir, 0444)
			Expect(err).NotTo(HaveOccurred())

			//Create a test file to process
			testFile := filepath.Join(sourceDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			err = processor.ProcessFiles()
			Expect(err).To(HaveOccurred()) // Should fail due to write permissions

			//Restore permissions for cleanup
			os.Chmod(outputDir, 0755)
		})

		It("should respect processing timeout", func() {
			// Use very short timeout
			shortTimeoutConfig := config
			shortTimeoutConfig.ProcessTimeout = 1 * time.Nanosecond

			shortProcessor, err := fileprocessor.NewFileProcessor(shortTimeoutConfig)
			Expect(err).NotTo(HaveOccurred())

			err = shortProcessor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if shortProcessor.IsRunning() {
					shortProcessor.Shutdown()
				}
			}()

			// Create test file
			testFile := filepath.Join(sourceDir, "timeout_test.txt")
			err = os.WriteFile(testFile, []byte("test content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			// Processing should handle timeout gracefully
			_ = shortProcessor.ProcessFiles()
			// Implementation should handle timeout without crashing
		})
	})

	// === PHASE 5: Concurrent Processing ===
	Describe("Concurrency", func() {
		BeforeEach(func() {
			var err error
			processor, err = fileprocessor.NewFileProcessor(config)
			Expect(err).NotTo(HaveOccurred())

			err = processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process multiple files concurrently", func() {
			//create multiple test files
			numFiles := 5

			for i := 0; i < numFiles; i++ {
				filename := filepath.Join(sourceDir, fmt.Sprintf("file_%d.txt", i))
				content := fmt.Sprintf("File %d content\nLine 2\nLine 3", i)
				err := os.WriteFile(filename, []byte(content), 0644)
				Expect(err).NotTo(HaveOccurred())
			}

			startTime := time.Now()
			err := processor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())
			processTime := time.Since(startTime)

			//Verify all files were processed
			resultFiles, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resultFiles)).To(Equal(numFiles))

			// Processing should be reasoably fast (concurrent processing)
			Expect(processTime).To(BeNumerically("<", 2*time.Second))
		})

		It("should handle queue overflow gracefully", func() {
			// Create more files than queue size
			smallQueueConfig := config
			smallQueueConfig.QueueSize = 2

			smallProcessor, err := fileprocessor.NewFileProcessor(smallQueueConfig)
			Expect(err).NotTo(HaveOccurred())

			err = smallProcessor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				if smallProcessor.IsRunning() {
					smallProcessor.Shutdown()
				}
			}()

			// Create files exceeding queue size
			numFiles := 10
			for i := 0; i < numFiles; i++ {
				filename := filepath.Join(sourceDir, fmt.Sprintf("overflow_%d.txt", i))
				err := os.WriteFile(filename, []byte("content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			}

			err = smallProcessor.ProcessFiles()
			Expect(err).NotTo(HaveOccurred())

			// All files should still be processed
			resultFiles, err := filepath.Glob(filepath.Join(outputDir, "*.json"))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(resultFiles)).To(Equal(numFiles))
		})
	})

	// === PHASE 6: Context Cancellation ===
	Describe("Context Handling", func() {
		BeforeEach(func() {
			var err error
			processor, err = fileprocessor.NewFileProcessor(config)
			Expect(err).NotTo(HaveOccurred())

			err = processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should respect context cancellation during processing", func() {
			// Create several test files
			for i := 0; i < 5; i++ {
				filename := filepath.Join(sourceDir, fmt.Sprintf("cancel_test_%d.txt", i))
				err := os.WriteFile(filename, []byte("content"), 0644)
				Expect(err).NotTo(HaveOccurred())
			}

			// Cancel context shortly after starting processing
			go func() {
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()

			err := processor.ProcessFiles()
			// Should handle cancellation gracefully
			Expect(err).To(HaveOccurred())
			Expect(processor.IsRunning()).To(BeFalse())
		})

	})
})
