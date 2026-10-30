package gpu

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
)

var ErrDuplicateLoader = errors.New("gpu loader already registered")

type Registry struct {
	mu      sync.RWMutex
	loaders map[string]Loader
}

func NewRegistry(loaders ...Loader) (*Registry, error) {
	registry := &Registry{loaders: make(map[string]Loader, len(loaders))}
	for _, loader := range loaders {
		if err := registry.Register(loader); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *Registry) Register(loader Loader) error {
	if loader == nil || (reflect.ValueOf(loader).Kind() == reflect.Pointer && reflect.ValueOf(loader).IsNil()) {
		return errors.New("gpu loader is nil")
	}
	vendor := loader.Vendor()
	if vendor == "" {
		return errors.New("gpu loader vendor is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.loaders[vendor]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateLoader, vendor)
	}
	if r.loaders == nil {
		r.loaders = make(map[string]Loader)
	}
	r.loaders[vendor] = loader
	return nil
}

func (r *Registry) Lookup(vendor string) (Loader, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	loader, ok := r.loaders[vendor]
	return loader, ok
}

func (r *Registry) Loaders() []Loader {
	r.mu.RLock()
	defer r.mu.RUnlock()
	loaders := make([]Loader, 0, len(r.loaders))
	for _, loader := range r.loaders {
		loaders = append(loaders, loader)
	}
	sort.Slice(loaders, func(i, j int) bool {
		return loaders[i].Vendor() < loaders[j].Vendor()
	})
	return loaders
}
