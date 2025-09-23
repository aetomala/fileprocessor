package fileprocessor_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFileprocessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fileprocessor Suite")
}
