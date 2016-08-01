package metric

import "sync"

type Store interface {
	Report(name string, value float64)
	Snapshot() map[string][]float64
}

func NewStore(maxCapacity int) Store {
	return &metricStore{
		lock:        &sync.Mutex{},
		data:        make(map[string][]float64),
		maxCapacity: maxCapacity,
	}
}

type metricStore struct {
	lock        *sync.Mutex
	data        map[string][]float64
	maxCapacity int
}

func (s *metricStore) Report(name string, value float64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	values := s.data[name]
	values = append(values, value)

	if len(values) > s.maxCapacity {
		values = values[s.maxCapacity/2:]
	}

	s.data[name] = values
}

func (s *metricStore) Snapshot() map[string][]float64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	ret := make(map[string][]float64)
	for k, v := range s.data {
		ret[k] = v
	}

	return ret
}
