// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package api_test

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"

	"github.com/gogo/protobuf/jsonpb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fuzz "github.com/google/gofuzz"
	"github.com/sqreen/go-agent/agent/internal/backend/api"
)

var fuzzer = fuzz.New().Funcs(FuzzStruct)

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

func TestMyStruct(t *testing.T) {
	var original api.Struct
	fuzzer.Fuzz(&original)

	buf, err := json.Marshal(original)
	require.NoError(t, err)
	t.Logf("original=%#v pb=%s", original, (string)(buf))

	var pb api.Struct
	err = json.Unmarshal(buf, &pb)
	require.NoError(t, err)

	require.Equal(t, pb, original)
}

func TestStruct(t *testing.T) {
	pb := &types.Struct{
		Fields: map[string]*types.Value{
			"field": &types.Value{
				Kind: &types.Value_StringValue{"a string"},
			},
		},
	}

	// Check it can be marshaled to the expected JSON struct.
	marshaler := &jsonpb.Marshaler{}
	str, err := marshaler.MarshalToString(pb)
	require.NoError(t, err)
	require.Equal(t, str, `{"field":"a string"}`)

	// Check it can be unmarshaled back to Protobuf.
	parsedPB := new(types.Struct)
	err = jsonpb.UnmarshalString(str, parsedPB)
	require.NoError(t, err)
	require.Equal(t, parsedPB, pb)
}

func FuzzStruct(e *api.Struct, c fuzz.Continue) {
	nbFields := c.Uint32() % 10
	if nbFields == 0 {
		e.Value = nil
		return
	}

	kv := make(map[string]interface{}, nbFields)
	e.Value = kv
	for n := 0; n < len(kv); n++ {
		var k string
		c.Fuzz(&k)

		var v interface{}
		switch c.Uint32() % 4 {
		case 0:
			v = nil
		case 1:
			var actual string
			c.Fuzz(&actual)
			v = actual
		case 2:
			var actual float64
			c.Fuzz(&actual)
			v = actual
		case 3:
			var actual bool
			c.Fuzz(&actual)
			v = actual
		}

		kv[k] = v
	}
}
