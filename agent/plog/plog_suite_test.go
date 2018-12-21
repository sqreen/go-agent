package plog_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPlog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plog Suite")
}
