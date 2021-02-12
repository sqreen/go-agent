// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package span_test

import (
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/sqreen/go-agent/internal/span"
	"github.com/stretchr/testify/require"
)

func TestUsage(t *testing.T) {
	seen := make(span.AttributeMap)

	onCreate := span.OnNewChildEventListenerFunc(func(s span.EmergingSpan) error {
		if v, ok := s.Get("span.name"); ok && v == "http.handler" {
			if v, ok := s.Get("k1"); ok {
				seen["k1"] = v
			}

			s.OnNewChild(func(s span.EmergingSpan) error {
				if v, ok := s.Get("k2"); ok {
					seen["k2"] = v
				} else if v, ok := s.Get("k3"); ok {
					seen["k3"] = v
				}
				return nil
			})
		}
		return nil
	})

	{
		rootSpan, _ := span.NewSpan(span.WithParent(span.RootSpan), span.WithEventListeners(&onCreate))
		{
			httpHandlerSpan, _ := span.NewSpan(span.WithParent(rootSpan), span.WithAttributes(span.AttributeMap{
				"span.name": "http.handler",
				"k1":        "v1",
			}))

			{
				subSpan, _ := span.NewSpan(span.WithParent(httpHandlerSpan), span.WithAttributes(span.AttributeMap{"k2": "v2"}))
				subSpan.End(nil)
			}

			{
				subSpan, _ := span.NewSpan(span.WithParent(httpHandlerSpan), span.WithAttributes(span.AttributeMap{"k3": "v3"}))
				subSpan.End(nil)
			}

			httpHandlerSpan.End(nil)
		}
		rootSpan.End(nil)
	}

	require.Equal(t, span.AttributeMap{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}, seen)
}

func TestUsage2(t *testing.T) {
	myOperation := func(parent span.Span, name string, v int, op func(s span.Span)) {
		sp, _ := span.NewSpan(span.WithParent(parent), span.WithAttributes(span.AttributeMap{
			"span.name":         name,
			name + ".attribute": v,
		}))
		defer sp.End(nil)
		op(sp)
	}

	onNewOp1Span := span.NewNamedChildSpanEventListener("my.operation.1", func(s span.EmergingSpan) error {
		s.OnNewNamedChild("my.operation.3", func(s span.EmergingSpan) error {
			var accumulator int

			s.OnNewNamedChild("my.operation.5", func(s span.EmergingSpan) error {
				v, exists := s.Get("my.operation.5.attribute")
				if !exists {
					return nil
				}
				accumulator += v.(int)
				return nil
			})

			s.OnEnd(func(results span.AttributeGetter) error {
				return s.EmitData(span.AttributeMap{"my.operation.3.accumulator": accumulator})
			})
			return nil
		})
		return nil
	})

	t.Run("expected stacking", func(t *testing.T) {
		seen := span.AttributeMap{}
		onData := span.OnChildDataEventListenerFunc(func(s span.Span, data span.AttributeGetter) error {
			if v, exists := data.Get("my.operation.3.accumulator"); exists {
				seen["my.operation.3.accumulator"] = v
			}
			return nil
		})

		rootSpan, _ := span.NewSpan(span.WithParent(span.RootSpan), span.WithEventListeners(onNewOp1Span, onData))
		myOperation(rootSpan, "my.operation.1", 1, func(s span.Span) {
			myOperation(s, "my.operation.2", 2, func(s span.Span) {
				myOperation(s, "my.operation.3", 3, func(s span.Span) {
					myOperation(s, "my.operation.4", 4, func(s span.Span) {
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
					})
				})
			})
		})
		rootSpan.End(nil)

		require.Equal(t, span.AttributeMap{
			"my.operation.3.accumulator": 25,
		}, seen)
	})

	t.Run("unexpected stacking", func(t *testing.T) {
		seen := span.AttributeMap{}
		onData := span.OnChildDataEventListenerFunc(func(s span.Span, data span.AttributeGetter) error {
			if v, exists := data.Get("my.operation.3.accumulator"); exists {
				seen["my.operation.3.accumulator"] = v
			}
			return nil
		})

		rootSpan, _ := span.NewSpan(span.WithParent(span.RootSpan), span.WithEventListeners(onNewOp1Span, onData))
		myOperation(rootSpan, "my.operation.0", 1, func(s span.Span) {
			myOperation(s, "my.operation.2", 2, func(s span.Span) {
				myOperation(s, "my.operation.3", 3, func(s span.Span) {
					myOperation(s, "my.operation.4", 4, func(s span.Span) {
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
					})
				})
			})
		})
		rootSpan.End(nil)

		require.Equal(t, span.AttributeMap{}, seen)
	})

	t.Run("unexpected stacking", func(t *testing.T) {
		seen := span.AttributeMap{}
		onData := span.OnChildDataEventListenerFunc(func(s span.Span, data span.AttributeGetter) error {
			if v, exists := data.Get("my.operation.3.accumulator"); exists {
				seen["my.operation.3.accumulator"] = v
			}
			return nil
		})

		rootSpan, _ := span.NewSpan(span.WithParent(span.RootSpan), span.WithEventListeners(onNewOp1Span, onData))
		myOperation(rootSpan, "my.operation.1", 1, func(s span.Span) {
			myOperation(s, "my.operation.2", 2, func(s span.Span) {
				myOperation(s, "my.operation.10", 3, func(s span.Span) {
					myOperation(s, "my.operation.4", 4, func(s span.Span) {
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
					})
				})
			})
		})
		rootSpan.End(nil)

		require.Equal(t, span.AttributeMap{}, seen)
	})

	t.Run("unexpected stacking", func(t *testing.T) {
		seen := span.AttributeMap{}
		onData := span.OnChildDataEventListenerFunc(func(s span.Span, data span.AttributeGetter) error {
			if v, exists := data.Get("my.operation.3.accumulator"); exists {
				seen["my.operation.3.accumulator"] = v
			}
			return nil
		})

		rootSpan, _ := span.NewSpan(span.WithParent(span.RootSpan), span.WithEventListeners(onNewOp1Span, onData))
		myOperation(rootSpan, "my.operation.3", 1, func(s span.Span) {
			myOperation(s, "my.operation.2", 2, func(s span.Span) {
				myOperation(s, "my.operation.1", 3, func(s span.Span) {
					myOperation(s, "my.operation.4", 4, func(s span.Span) {
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
						myOperation(s, "my.operation.5", 5, func(s span.Span) {})
					})
				})
			})
		})
		rootSpan.End(nil)

		require.Equal(t, span.AttributeMap{}, seen)
	})

}

func TestUsage3(t *testing.T) {
	myOperation := func(parent span.Span, name string, startAttrs span.AttributeMap, op func(parent span.Span) span.AttributeMap) {
		if startAttrs == nil {
			startAttrs = span.AttributeMap{}
		}
		startAttrs["span.name"] = name
		s, _ := span.NewSpan(span.WithParent(parent), span.WithAttributes(startAttrs))
		results := op(s)
		s.End(results)
	}

	seen := span.AttributeMap{}

	onNewHTTPHandlerSpan := span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
		subscriptions := []string{"server.request.body.raw", "server.request.body",
			"server.request.url.query", "server.request.url.raw_query"}
		s.OnChildData(func(s span.Span, data span.AttributeGetter) error {
			for _, name := range subscriptions {
				if v, exists := data.Get(name); exists {
					seen[name] = v
				}
			}
			return nil
		})

		if rawQuery, exists := s.Get("server.request.url.raw_query"); exists {
			s.OnNewNamedChild("url.parse_query", func(s span.EmergingSpan) error {
				funcValue, _ := s.Get("url.parse_query.raw_query")

				if funcValue != rawQuery {
					return nil
				}

				s.OnEnd(func(results span.AttributeGetter) error {
					v, exists := results.Get("url.query.values")
					if !exists {
						return nil
					}
					values, ok := v.(url.Values)
					if !ok {
						return nil
					}
					return s.EmitData(span.AttributeMap{"server.request.url.query": values})
				})
				return nil
			})

			s.OnNewNamedChild("json.decode", func(s span.EmergingSpan) error {
				var fullbody []byte

				s.OnNewNamedChild("request.body.read", func(s span.EmergingSpan) error {
					s.OnEnd(func(results span.AttributeGetter) error {
						v, exists := results.Get("request.body.read.buffer")
						if !exists {
							return nil
						}

						fullbody = append(fullbody, v.([]byte)...)
						return nil
					})
					return nil
				})

				s.OnEnd(func(results span.AttributeGetter) error {
					jsonValue, _ := results.Get("json.decode.value")
					return s.EmitData(span.AttributeMap{
						"server.request.body.raw": fullbody,
						"server.request.body":     jsonValue,
					})
				})
				return nil
			})
		}
		return nil
	})

	rootSpan, _ := span.NewSpan(span.WithParent(span.RootSpan), span.WithEventListeners(onNewHTTPHandlerSpan))
	myOperation(rootSpan, "http.handler", span.AttributeMap{"server.request.url.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) span.AttributeMap {
		myOperation(s, "url.parse_query", span.AttributeMap{"url.parse_query.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) span.AttributeMap {
			return span.AttributeMap{"url.query.values": url.Values{"k1": []string{"v11", "v12"}, "k2": []string{"v2"}}}
		})

		myOperation(s, "json.decode", nil, func(s span.Span) span.AttributeMap {
			myOperation(s, "request.body.read", nil, func(s span.Span) span.AttributeMap { return span.AttributeMap{"request.body.read.buffer": []byte("one")} })
			myOperation(s, "request.body.read", nil, func(s span.Span) span.AttributeMap { return span.AttributeMap{"request.body.read.buffer": []byte("two")} })
			myOperation(s, "request.body.read", nil, func(s span.Span) span.AttributeMap { return span.AttributeMap{"request.body.read.buffer": []byte("three")} })
			myOperation(s, "request.body.read", nil, func(s span.Span) span.AttributeMap { return span.AttributeMap{"request.body.read.buffer": []byte("four")} })
			myOperation(s, "request.body.read", nil, func(s span.Span) span.AttributeMap { return span.AttributeMap{"request.body.read.buffer": []byte("five")} })

			return span.AttributeMap{"json.decode.value": []interface{}{"a", "json", "array"}}
		})

		return nil
	})
	rootSpan.End(nil)

	require.Equal(t, span.AttributeMap{
		"server.request.body":      []interface{}{"a", "json", "array"},
		"server.request.body.raw":  []byte("onetwothreefourfive"),
		"server.request.url.query": url.Values{"k1": []string{"v11", "v12"}, "k2": []string{"v2"}},
	}, seen)
}

func TestUsage4(t *testing.T) {
	myOperation := func(parent span.Span, name string, startAttrs span.AttributeMap, op func(parent span.Span) (span.AttributeMap, error)) error {
		if startAttrs == nil {
			startAttrs = span.AttributeMap{}
		}
		startAttrs["span.name"] = name

		s, err := span.NewSpan(span.WithParent(parent), span.WithAttributes(startAttrs))
		if err != nil {
			return err
		}
		results, err := op(s)
		if err != nil {
			s.End(results)
			return err
		}
		return s.End(results)
	}

	requestURLQueryParsingListener := span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
		rawURLQuery, exists := s.Get("server.request.url.raw_query")
		if !exists {
			// eg. when not yet completely instrumented
			return nil
		}

		s.OnNewNamedChild("url.parse_query", func(s span.EmergingSpan) error {
			funcValue, exists := s.Get("url.parse_query.raw_query")
			if !exists || funcValue != rawURLQuery {
				return nil
			}

			s.OnEnd(func(results span.AttributeGetter) error {
				v, exists := results.Get("url.query.values")
				if !exists {
					return nil
				}
				return s.EmitData(span.AttributeMap{"server.request.url.query": v})
			})
			return nil
		})
		return nil
	})

	requestJSONBodyParsingListener := span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
		s.OnNewNamedChild("json.decode", func(s span.EmergingSpan) error {
			var fullbody []byte

			s.OnNewNamedChild("request.body.read", func(s span.EmergingSpan) error {
				s.OnChildData(func(s span.Span, data span.AttributeGetter) error {
					v, exists := data.Get("request.body.read.buffer")
					if !exists {
						return nil
					}
					fullbody = append(fullbody, v.([]byte)...)
					return nil
				})
				return nil
			})

			s.OnEnd(func(results span.AttributeGetter) error {
				jsonValue, _ := results.Get("json.decode.value")
				return s.EmitData(span.AttributeMap{
					"server.request.body.raw": string(fullbody),
					"server.request.body":     jsonValue,
				})
			})
			return nil
		})
		return nil
	})

	wafProtection := span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
		subscriptions := []string{
			"server.request.body.raw",
			"server.request.body",
			"server.request.url.query",
		}

		cache := map[string]interface{}{}

		runWAF := func() error {
			for _, v := range cache {
				if v == "attack" {
					return errors.New("waf attack detected")
				}
			}
			return nil
		}

		s.OnNewChild(func(s span.EmergingSpan) error {
			for _, subscription := range subscriptions {
				if v, exists := s.Get(subscription); exists {
					cache[subscription] = v
				}
			}
			return runWAF()
		})

		s.OnChildData(func(s span.Span, data span.AttributeGetter) error {
			for _, subscription := range subscriptions {
				if v, exists := data.Get(subscription); exists {
					cache[subscription] = v
				}
			}
			return runWAF()
		})
		return nil
	})

	sqliProtection := span.NewNamedChildSpanEventListener("http.handler", func(s span.EmergingSpan) error {
		subscriptions := map[string]struct{}{
			"server.request.body.raw":  {},
			"server.request.body":      {},
			"server.request.url.query": {},
		}

		cache := map[string]interface{}{}

		s.OnNewChild(func(s span.EmergingSpan) error {
			for subscription := range subscriptions {
				if v, exists := s.Get(subscription); exists {
					cache[subscription] = v
				}
			}
			return nil
		})

		s.OnChildData(func(s span.Span, data span.AttributeGetter) error {
			for subscription := range subscriptions {
				if v, exists := data.Get(subscription); exists {
					cache[subscription] = v
				}
			}
			return nil
		})

		s.OnNewNamedChild("sql.query", func(s span.EmergingSpan) error {
			s.OnEnd(func(results span.AttributeGetter) error {
				v, _ := s.Get("sql.query.value")
				query := v.(string)
				for _, v := range cache {
					substr, ok := v.(string)
					if !ok {
						continue
					}
					if strings.Contains(query, substr) {
						return errors.New("sqli attack detected")
					}
				}
				return nil
			})
			return nil
		})
		return nil
	})

	t.Run("stack 1", func(t *testing.T) {
		rootSpan, err := span.NewSpan(span.WithParent(nil), span.WithEventListeners(requestURLQueryParsingListener, requestJSONBodyParsingListener, wafProtection))
		require.NoError(t, err)
		err = myOperation(rootSpan, "http.handler", span.AttributeMap{"server.request.url.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) (span.AttributeMap, error) {
			err := myOperation(s, "url.parse_query", span.AttributeMap{"url.parse_query.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) (span.AttributeMap, error) {
				return span.AttributeMap{"url.query.values": url.Values{"k1": []string{"v11", "v12"}, "k2": []string{"v2"}}}, nil
			})
			if err != nil {
				return nil, err
			}

			err = myOperation(s, "json.decode", nil, func(s span.Span) (span.AttributeMap, error) {
				err := myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("one")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("two")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("three")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("four")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("five")}, nil })
				if err != nil {
					return nil, err
				}

				return span.AttributeMap{"json.decode.value": "attack"}, nil
			})
			if err != nil {
				return nil, err
			}

			return nil, myOperation(s, "sql.query", span.AttributeMap{"sql.query.value": "SELECT * FROM users"}, func(s span.Span) (span.AttributeMap, error) { return nil, nil })
		})
		rootSpan.End(nil)

		require.Error(t, err)
		require.Equal(t, "waf attack detected", err.Error())
	})

	t.Run("stack 2", func(t *testing.T) {
		rootSpan, err := span.NewSpan(span.WithParent(nil), span.WithEventListeners(requestURLQueryParsingListener, requestJSONBodyParsingListener, wafProtection))
		require.NoError(t, err)
		err = myOperation(rootSpan, "http.handler", span.AttributeMap{"server.request.url.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) (span.AttributeMap, error) {
			err := myOperation(s, "url.parse_query", span.AttributeMap{"url.parse_query.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) (span.AttributeMap, error) {
				return span.AttributeMap{"url.query.values": url.Values{"k1": []string{"v11", "v12"}, "k2": []string{"v2"}}}, nil
			})
			if err != nil {
				return nil, err
			}

			err = myOperation(s, "json.decode", nil, func(s span.Span) (span.AttributeMap, error) {
				err := myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("one")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("two")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("three")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("four")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("five")}, nil })
				if err != nil {
					return nil, err
				}

				return span.AttributeMap{"json.decode.value": []interface{}{"a", "json", "array"}}, nil
			})
			if err != nil {
				return nil, err
			}

			return nil, myOperation(s, "sql.query", span.AttributeMap{"sql.query.value": "SELECT * FROM users"}, func(s span.Span) (span.AttributeMap, error) { return nil, nil })
		})
		rootSpan.End(nil)

		require.NoError(t, err)
	})

	t.Run("stack 3", func(t *testing.T) {
		rootSpan, err := span.NewSpan(span.WithParent(nil), span.WithEventListeners(requestURLQueryParsingListener, requestJSONBodyParsingListener, wafProtection, sqliProtection))
		require.NoError(t, err)
		err = myOperation(rootSpan, "http.handler", span.AttributeMap{"server.request.url.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) (span.AttributeMap, error) {
			err := myOperation(s, "url.parse_query", span.AttributeMap{"url.parse_query.raw_query": "k1=v11&k1=v12&k2=v2"}, func(s span.Span) (span.AttributeMap, error) {
				return span.AttributeMap{"url.query.values": url.Values{"k1": []string{"v11", "v12"}, "k2": []string{"v2"}}}, nil
			})
			if err != nil {
				return nil, err
			}

			err = myOperation(s, "json.decode", nil, func(s span.Span) (span.AttributeMap, error) {
				err := myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte("a")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte(" body")}, nil })
				if err != nil {
					return nil, err
				}
				err = myOperation(s, "request.body.read", nil, func(s span.Span) (span.AttributeMap, error) { return span.AttributeMap{"request.body.read.buffer": []byte(" injection")}, nil })
				if err != nil {
					return nil, err
				}

				return span.AttributeMap{"json.decode.value": []interface{}{"a", "json", "array"}}, nil
			})
			if err != nil {
				return nil, err
			}

			return nil, myOperation(s, "sql.query", span.AttributeMap{"sql.query.value": "SELECT * FROM users WHERE a body injection"}, func(s span.Span) (span.AttributeMap, error) { return nil, nil })
		})
		rootSpan.End(nil)

		require.Error(t, err)
		require.Equal(t, "sqli attack detected", err.Error())
	})
}
