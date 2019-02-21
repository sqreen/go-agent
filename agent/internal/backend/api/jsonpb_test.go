package api_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
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
			buf, err := json.Marshal(pb)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf)).To(Equal(`{"batch":[{"event_type":"request_record","A":16,"B":22}]}`))
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
					buf, err := json.Marshal(pb)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(buf)).To(Equal(`["my key","my value"]`))
				})
			})
		})

		Describe("Observed", func() {
			Describe("SDK events", func() {
				var (
					pb       *api.RequestRecord_Observed_SDKEvent_Args
					str      string
					err      error
					expected string
				)

				JustBeforeEach(func() {
					var buf []byte
					buf, err = json.Marshal(pb)
					str = string(buf)
				})

				Describe("Track event", func() {

					Context("with properties", func() {
						BeforeEach(func() {
							expected = `["my event",{"properties":{"key 1":33,"key 2":"value 2","key 3":[1,2,3],"key 4":{"A":16,"B":22}}}]`
							pb = &api.RequestRecord_Observed_SDKEvent_Args{
								Args: &api.RequestRecord_Observed_SDKEvent_Args_Track_{
									Track: &api.RequestRecord_Observed_SDKEvent_Args_Track{
										Event: "my event",
										Options: &api.RequestRecord_Observed_SDKEvent_Args_Track_Options{
											Properties: &api.Struct{
												map[string]interface{}{
													"key 1": 33,
													"key 2": "value 2",
													"key 3": []int{1, 2, 3},
													"key 4": struct{ A, B int }{A: 16, B: 22},
												},
											},
										},
									},
								},
							}
						})

						It("should marshal to an event object", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(str).To(Equal(expected))
						})
					})

					Context("user identifiers", func() {
						BeforeEach(func() {
							expected = `["my event",{"user_identifiers":{"key 1":33,"key 2":"value 2","key 3":[1,2,3],"key 4":{"A":16,"B":22}}}]`
							pb = &api.RequestRecord_Observed_SDKEvent_Args{
								Args: &api.RequestRecord_Observed_SDKEvent_Args_Track_{
									Track: &api.RequestRecord_Observed_SDKEvent_Args_Track{
										Event: "my event",
										Options: &api.RequestRecord_Observed_SDKEvent_Args_Track_Options{
											UserIdentifiers: &api.Struct{
												map[string]interface{}{
													"key 1": 33,
													"key 2": "value 2",
													"key 3": []int{1, 2, 3},
													"key 4": struct{ A, B int }{A: 16, B: 22},
												},
											},
										},
									},
								},
							}
						})

						It("should marshal to an event object", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(str).To(Equal(expected))
						})
					})

					Context("without options", func() {
						BeforeEach(func() {
							expected = `["my event"]`
							pb = &api.RequestRecord_Observed_SDKEvent_Args{
								Args: &api.RequestRecord_Observed_SDKEvent_Args_Track_{
									Track: &api.RequestRecord_Observed_SDKEvent_Args_Track{
										Event: "my event",
									},
								},
							}
						})

						It("should marshal to an event object", func() {
							Expect(err).NotTo(HaveOccurred())
							Expect(str).To(Equal(expected))
						})
					})
				})

				Describe("Identify event", func() {
					It("should marshal to an event object", func() {
						pb := &api.RequestRecord_Observed_SDKEvent_Args{
							Args: &api.RequestRecord_Observed_SDKEvent_Args_Identify_{
								Identify: &api.RequestRecord_Observed_SDKEvent_Args_Identify{
									UserIdentifiers: &api.Struct{
										map[string]interface{}{
											"key 1": 33,
											"key 2": "value 2",
											"key 3": []int{1, 2, 3},
											"key 4": struct{ A, B int }{A: 16, B: 22},
										},
									},
								},
							},
						}

						buf, err := json.Marshal(pb)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(buf)).To(Equal(`[{"key 1":33,"key 2":"value 2","key 3":[1,2,3],"key 4":{"A":16,"B":22}}]`))
					})
				})
			})
		})
	})
})
