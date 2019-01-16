package sdk_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/tools/testlib"
)

func performSDKCalls(ctx *sdk.HTTPRequestContext) func() {
	return func() {
		event := ctx.Track(testlib.RandString(0, 50))
		Expect(event).To(BeNil())
		event = event.WithTimestamp(time.Now())
		Expect(event).To(BeNil())
		event = event.WithProperties(nil)
		Expect(event).To(BeNil())
		event = ctx.Track(testlib.RandString(0, 50))
		Expect(event).To(BeNil())
		event = event.WithProperties(nil)
		Expect(event).To(BeNil())
		event = event.WithTimestamp(time.Now())
		Expect(event).To(BeNil())
		ctx.Close()
	}
}

var _ = Describe("SDK", func() {
	var ctx *sdk.HTTPRequestContext
	Context("when used with zero value (nil)", func() {
		It("should not panic", func() {
			Expect(performSDKCalls(ctx)).ToNot(Panic())
		})
	})
	Context("when used with zero value (nil)", func() {
		It("should not panic", func() {
			Expect(performSDKCalls(ctx)).ToNot(Panic())
		})
	})
})
