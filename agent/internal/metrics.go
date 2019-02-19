package internal

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sqreen/go-agent/agent/internal/backend/api"
	"github.com/sqreen/go-agent/agent/internal/config"
)

type metricsManager struct {
	ctx       context.Context
	metrics   sync.Map
	readyLock sync.Mutex
	ready     []api.MetricResponse
}

func newMetricsManager(ctx context.Context) *metricsManager {
	return &metricsManager{
		ctx: ctx,
	}
}

type metricsStore struct {
	done     func(start, finish time.Time, observations sync.Map)
	period   time.Duration
	entries  sync.Map
	once     sync.Once
	swapLock sync.RWMutex
	expired  bool
}

type metricEntry interface {
	// Deterministic marshaling if possible...
	bucketID() (string, error)
}

func (m *metricsManager) get(name string) *metricsStore {
	store := &metricsStore{
		period: time.Minute,
		done: func(start, finish time.Time, observations sync.Map) {
			m.metrics.Delete(name)
			logger.Debug("metrics ", name, " ready")
			m.addObservations(name, start, finish, observations)
		},
	}

	actual, _ := m.metrics.LoadOrStore(name, store)
	store = actual.(*metricsStore)
	store.once.Do(func() {
		go func() {
			logger.Debug("bookkeeping metrics ", name, " with period ", store.period)
			store.monitor(m.ctx, time.Now())
		}()
	})

	return store
}

func (m *metricsManager) addObservations(name string, start, finish time.Time, observations sync.Map) {
	observation := make(map[string]uint64)
	observations.Range(func(k, v interface{}) bool {
		key, ok := k.(string)
		if !ok {
			logger.Panic(errors.New("unexpected metric key type"))
			return true
		}

		value, ok := v.(*uint64)
		if !ok {
			logger.Panic(errors.New("unexpected metric value type"))
			return true
		}

		observation[key] = *value
		return true
	})

	metric := api.MetricResponse{
		Name:        name,
		Start:       start,
		Finish:      finish,
		Observation: api.Struct{observation},
	}

	m.readyLock.Lock()
	defer m.readyLock.Unlock()
	m.ready = append(m.ready, metric)
}

func (m *metricsManager) getObservations() []api.MetricResponse {
	m.readyLock.Lock()
	defer m.readyLock.Unlock()
	ready := m.ready
	m.ready = m.ready[0:0]
	return ready
}

func (s *metricsStore) add(e metricEntry) {
	s.swapLock.RLock()
	defer s.swapLock.RUnlock()

	if s.expired {
		// FIXME: better design preventing this case
		// For now, a few events may be dropped.
		return
	}

	var n uint64 = 1
	key, err := e.bucketID()
	if err != nil {
		logger.Error("could not compute the bucket id of the metric key:", err)
		return
	}
	actual, loaded := s.entries.LoadOrStore(key, &n)
	if loaded {
		newVal := atomic.AddUint64(actual.(*uint64), 1)
		logger.Debug("metric store ", key, " set to ", newVal)
	} else {
		logger.Debug("metric store ", key, " set to ", n)
	}
}

func (s *metricsStore) monitor(ctx context.Context, start time.Time) {
	var finish time.Time
	select {
	case <-ctx.Done():
		finish = time.Now()
	case finish = <-time.After(s.period):
	}

	s.swapLock.Lock()
	entries := s.entries
	s.entries = sync.Map{}
	s.expired = true
	s.swapLock.Unlock()

	s.done(start, finish, entries)
}

func addUserEvent(event userEventFace) {
	if config.Disable() || metricsMng == nil {
		// Disabled or not yet initialized agent
		return
	}

	var store *metricsStore
	switch actual := event.(type) {
	case *authUserEvent:
		if actual.loginSuccess {
			store = metricsMng.get("sdk-login-success")
		} else {
			store = metricsMng.get("sdk-login-fail")
		}
	case *signupUserEvent:
		store = metricsMng.get("sdk-signup")
	}

	store.add(event)
}
