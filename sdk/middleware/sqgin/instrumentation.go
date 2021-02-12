// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sqgin

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime/trace"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	http_protection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/rule"
	"github.com/sqreen/go-agent/internal/rule/callback"
	"github.com/sqreen/go-agent/internal/span"
	"github.com/sqreen/go-agent/internal/sqlib/sqhook"
)

func init() {
	rule.Register(
		rule.Descriptor{
			Name: "http request body reading",
			Instrumentation: rule.NativeInstrumentation{
				Function: "net/http.(*body).Read",
				Callback: func(params []reflect.Value) (epilog sqhook.ReflectedEpilogCallback, prologErr error) {
					if s := span.Current(); s == nil || span.ProtectionContext(s) == nil {
						return nil, nil
					}

					// func (b *body) Read(p []byte) (n int, err error)
					sp, err := span.NewSpan(span.WithAttributes(span.AttributeMap{"span.name": "http.request.body.read"}))
					if err != nil {
						epilog = callback.MakeBlockingEpilog(1, err)
						prologErr = sqhook.AbortError
						return
					}

					epilog = func(results []reflect.Value) {
						n := results[0].Elem().Interface().(int)
						err := results[1].Elem().Interface().(error)

						var buf []byte
						if n > 0 {
							p := params[1].Elem().Interface().([]byte)
							buf = p[:n]
						}

						if err := sp.End(span.AttributeMap{
							"buffer": buf,
							"error":  err,
						}); err != nil {
							callback.ApplyBlockingError(results, 1, err)
							return
						}
					}
					return
				},
			},
		},

		rule.Descriptor{
			Name: "gin request data binding",
			Instrumentation: rule.NativeInstrumentation{
				Function: "github.com/gin-gonic/gin.(*Context).ShouldBindWith",
				Callback: func(c **gin.Context, obj *interface{}, b *binding.Binding) (epilog func(*error), prologErr error) {
					sp, err := span.NewSpan(span.WithAttributes(span.AttributeMap{"span.name": "gin.context.bind"}))
					if err != nil {
						epilog = func(fnErr *error) {
							*fnErr = err
						}
						prologErr = sqhook.AbortError
						return
					}

					epilog = func(fnErr *error) {
						if *fnErr != nil {
							_ = sp.End(nil)
							return
						}
						*fnErr = sp.End(span.AttributeMap{"value": *obj})
					}
					return
				},
			},
		},

		rule.Descriptor{
			Name: "gin request body value producer",
			Instrumentation: rule.SpanInstrumentation{
				EventListener: span.NewNamedChildSpanEventListener("gin.context.bind", func(s span.EmergingSpan) error {
					var didBodyRead bool

					s.OnNewNamedChild("http.request.body.read", func(s span.EmergingSpan) error {
						didBodyRead = true
						return nil
					})

					s.OnEnd(func(results span.AttributeGetter) error {
						// Dispatch body addresses if a body has been read - meaning that
						// gin's bind function was used to parse the body.

						if results == nil || !didBodyRead {
							return nil
						}

						value, exists := results.Get("value")
						if !exists {
							return nil
						}

						return s.EmitData(span.AttributeMap{
							"server.request.body": value,
						})
					})
					return nil
				}),
			},
		},

		rule.Descriptor{
			Name: "http request body producer",
			Instrumentation: rule.SpanInstrumentation{
				EventListener: span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
					var (
						mu      sync.Mutex
						rawbody []byte
					)

					s.OnNewNamedChild("http.request.body.read", func(s span.EmergingSpan) error {
						s.OnEnd(func(results span.AttributeGetter) (onEndError error) {
							// Append the resulting read buffer

							if results == nil {
								return nil
							}

							v, exists := results.Get("error")
							if !exists {
								return nil
							}
							err := v.(error)

							if err == io.EOF {
								defer func() {
									onEndError = s.EmitData(span.AttributeMap{"server.request.body.raw": rawbody})
								}()
							}

							v, exists = results.Get("buffer")
							if !exists {
								return nil
							}

							buf := v.([]byte)
							if len(buf) == 0 {
								return nil
							}

							mu.Lock()
							defer mu.Unlock()
							rawbody = append(rawbody, buf...)

							return nil
						})
						return nil
					})
					return nil
				}),
			},
		},
	)

	rule.Register(rule.Descriptor{
		Name: "http request multipart form data parser",
		Instrumentation: rule.NativeInstrumentation{
			Function: "net/http.(*Request).ParseMultipartForm",
			Callback: func(r **http.Request, maxMemory *int64) (epilog func(*error), prologErr error) {
				// func (r *Request) ParseMultipartForm(maxMemory int64) error

				sp, err := span.NewSpan(span.WithAttributes(span.AttributeMap{
					"span.name": "http.request.parse_multipart_form",
					"max_size":  *maxMemory,
				}))

				if err != nil {
					epilog = func(fnErr *error) {
						*fnErr = err
					}
					prologErr = sqhook.AbortError
					return
				}

				epilog = func(fnErr *error) {
					var results span.AttributeMap
					// No matter what, we must end the span
					defer func() {
						spErr := sp.End(results)
						if *fnErr == nil {
							*fnErr = spErr
						}
					}()

					if *fnErr != nil {
						return
					}

					data := (*r).MultipartForm
					formNames := make([]string, 0, len(data.Value))
					for name := range data.Value {
						formNames = append(formNames, name)
					}

					var (
						combinedSize int64
						filenames    = make([]string, 0, len(data.File))
					)
					for filename, fileHeaders := range data.File {
						filenames = append(filenames, filename)
						for _, fh := range fileHeaders {
							combinedSize += fh.Size
						}
					}

					results = span.AttributeMap{
						"server.request.body.files_field_names":  formNames,
						"server.request.body.filenames":          filenames,
						"server.request.body.combined_file_size": combinedSize,
					}

					*fnErr = sp.EmitData(results)
				}
				return
			},
		},
	})

	rule.Register(rule.Descriptor{
		Name: "http request post form data parser",
		Instrumentation: rule.NativeInstrumentation{
			Function: "net/http.(*Request).ParseForm",
			Callback: func(r **http.Request) (epilog func(*error), prologErr error) {
				// func (r *Request) ParseForm() error

				sp, err := span.NewSpan(span.WithAttributes(span.AttributeMap{
					"span.name": "http.request.parse_form",
				}))

				if err != nil {
					epilog = func(fnErr *error) {
						*fnErr = err
					}
					prologErr = sqhook.AbortError
					return
				}

				epilog = func(fnErr *error) {
					var results span.AttributeMap
					// No matter what, we must end the span
					defer func() {
						if err := sp.End(results); *fnErr == nil {
							*fnErr = err
						}
					}()

					if *fnErr != nil {
						return
					}

					postFormData := map[string][]string((*r).PostForm)
					results = span.AttributeMap{
						"server.request.body": postFormData,
					}

					*fnErr = sp.EmitData(results)
				}
				return
			},
		},
	})

	// Binding accessor data bridge for now
	rule.Register(rule.Descriptor{
		Name: "binding accessor request parameters feed",
		Instrumentation: rule.SpanInstrumentation{
			EventListener: span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
				p := span.ProtectionContext(s)
				if p == nil {
					return nil
				}
				var mu sync.Mutex

				setValue := func(address string, value interface{}) {
					mu.Lock()
					defer mu.Unlock()
					p.SetRequestBindingAccessorValue(address, value)
				}

				wanted := []string{
					"server.request.query",
					"server.request.body",
					"server.request.body.raw",
					"server.request.path_params",
				}

				// TODO: set once

				for _, wanted := range wanted {
					if v, exists := s.Get(wanted); exists {
						setValue(wanted, v)
					}
				}

				s.OnChildData(func(s span.Span, data span.AttributeGetter) error {
					for _, wanted := range wanted {
						if v, exists := data.Get(wanted); exists {
							setValue(wanted, v)
						}
					}
					return nil
				})
				return nil
			}),
		},
	})

	rule.Register(
		rule.Descriptor{
			Name: "url query parsing",
			Instrumentation: rule.NativeInstrumentation{
				Function: "net/url.ParseQuery",
				Callback: func(query *string) (epilog func(*url.Values, *error), prologErr error) {
					// func ParseQuery(query string) (Values, error)
					sp, err := span.NewSpan(span.WithAttributes(span.AttributeMap{
						"span.name": "net.url.parse_query",
						"query":     *query,
					}))

					if err != nil {
						epilog = func(_ *url.Values, fnErr *error) {
							*fnErr = err
						}
						prologErr = sqhook.AbortError
						return
					}

					epilog = func(values *url.Values, fnErr *error) {
						var attrs span.AttributeMap
						if *fnErr == nil {
							attrs = span.AttributeMap{
								"values": map[string][]string(*values),
							}
						}
						*fnErr = sp.End(attrs)
					}
					return
				},
			},
		},

		rule.Descriptor{
			Name: "http request url query parsing",
			Instrumentation: rule.SpanInstrumentation{
				EventListener: span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
					v, exists := s.Get("server.request.go_url")
					if !exists {
						return nil
					}

					u, ok := v.(*url.URL)
					if !ok {
						return nil
					}

					var (
						rawQuery = u.RawQuery
					)

					s.OnNewNamedChild("net.url.parse_query", func(s span.EmergingSpan) error {
						v, exists := s.Get("query")
						if !exists {
							return nil
						}

						query, ok := v.(string)
						if !ok {
							return nil
						}

						if query != rawQuery {
							// We are not parsing the request query string
							return nil
						}

						s.OnEnd(func(results span.AttributeGetter) error {
							if results == nil {
								return nil
							}

							values, exists := results.Get("values")
							if !exists {
								return nil
							}

							return s.EmitData(span.AttributeMap{"server.request.query": values})
						})
						return nil
					})
					return nil
				}),
			},
		},
	)

	rule.Register(rule.Descriptor{
		Name: "go trace",
		Instrumentation: rule.SpanInstrumentation{
			EventListener: span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
				ctx, task := trace.NewTask(context.Background(), "http.handler")

				s.OnEnd(func(results span.AttributeGetter) error {
					task.End()
					return nil
				})

				s.OnNewChild(func(s span.EmergingSpan) error {
					v, exists := s.Get("span.name")
					if !exists {
						return nil
					}
					name := v.(string)
					region := trace.StartRegion(ctx, name)
					s.OnEnd(func(results span.AttributeGetter) error {
						region.End()
						return nil
					})
					return nil
				})
				return nil
			}),
		},
	})

	rule.Register(rule.Descriptor{
		Name: "http status code monitoring",
		Instrumentation: rule.SpanInstrumentation{
			EventListener: span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
				v := span.ProtectionContext(s)
				if v == nil {
					return nil
				}

				p := v.(*http_protection.ProtectionContext)
				if p == nil {
					return nil
				}

				s.OnEnd(func(results span.AttributeGetter) error {
					response := results.(types.ResponseFace)
					if response != nil {
						p.MonitorObservedResponse(response)
					}
					return nil
				})

				return nil
			}),
		},
	})

}
