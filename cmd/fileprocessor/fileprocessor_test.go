package fileprocessor_test

import (
	"context"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"fileprocessor/cmd/fileprocessor"
)

var _ = Describe("Fileprocessor", func() {
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

		//Create temporary directories for testing
		var err error
		tempDir, err = os.MkdirTemp("", "fileprocessor_test")
		Expect(err).NotTo(HaveOccurred())

		sourceDir = filepath.Join(tempDir, "source")
		outputDir = filepath.Join(tempDir, "output")

		err = os.MkdirAll(sourceDir, 0755)
		Expect(err).NotTo(HaveOccurred())
		err = os.MkdirAll(outputDir, 0755)

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
		It("should create a new FileProcessor with a valid configuration", func() {
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
				processor, err := fileprocessor.NewFileProcessor(cfg)
				Expect(err).To(HaveOccurred())
				Expect(processor).To(BeNil())
			}
		})

		It("should apply default values for optional configuration", func() {
			minimalConfig := fileprocessor.Config{
				SourceDir: sourceDir,
				OutputDir: outputDir,
			}

			var err error
			processor, err := fileprocessor.NewFileProcessor(minimalConfig)

			Expect(err).NotTo(HaveOccurred())
			Expect(processor).NotTo(BeNil())
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

		It("should validate directories exist at statup", func() {
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
			cancelFunc() //Cancel immediatelly

			err := processor.Start(cancelledCtx)
			Expect(err).To(HaveOccurred())
			Expect(processor.IsRunning()).To(BeFalse())
		})

		It("should maintain isRunning state correctly", func() {
			Expect(processor.IsRunning()).To(BeFalse())

			err := processor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(processor.IsRunning()).To(BeTrue())
		})
	})

})
