package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sqreen/go-agent/agent/backend/api"
)

var _ = Describe("API", func() {

	Describe("Batch", func() {
		It("should successfully marshal to json", func() {
			pb := &api.BatchRequest{
				Batch: []api.BatchRequest_Event{
					{
						EventType: "request_record",
						Event: api.Struct{
							&struct{ A, B int }{A: 16, B: 22},
						},
					},
				},
			}
			str, err := api.DefaultJSONPBMarshaler.MarshalToString(pb)
			Expect(err).NotTo(HaveOccurred())
			Expect(str).To(Equal(`{"batch":[{"event_type":"request_record","A":16,"B":22}]}`))
		})
	})

	Describe("Request Record", func() {
		Describe("Request", func() {
			Describe("Headers", func() {
				It("should marshal a header to a two-element array", func() {
					pb := &api.RequestRecord_Request_Header{
						Key:   "my key",
						Value: "my value",
					}
					str, err := api.DefaultJSONPBMarshaler.MarshalToString(pb)
					Expect(err).NotTo(HaveOccurred())
					Expect(str).To(Equal(`["my key", "my value"]`))
				})
			})
		})

		Describe("Observed", func() {
			Describe("SDK events", func() {
				It("should marshal to an event object", func() {
					pb := api.ListValue([]interface{}{
						"my event",
						&api.RequestRecord_Observed_SDKEvent_Options{
							Properties: &api.Struct{
								map[string]interface{}{
									"key 1": 33,
									"key 2": "value 2",
									"key 3": []int{1, 2, 3},
									"key 4": struct{ A, B int }{A: 16, B: 22},
								},
							},
						},
					})
					str, err := api.DefaultJSONPBMarshaler.MarshalToString(pb)
					Expect(err).NotTo(HaveOccurred())
					Expect(str).To(Equal(`["my event",{"properties":{"key 1":33,"key 2":"value 2","key 3":[1,2,3],"key 4":{"A":16,"B":22}}}]`))
				})
			})
		})
	})

})
