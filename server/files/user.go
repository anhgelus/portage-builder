package files

import (
	"context"
	"errors"
	"sync"
)

type Manager struct {
	mu     sync.RWMutex
	data   map[string]*Root
	folder string
}

func NewManager(folder string) *Manager {
	return &Manager{data: make(map[string]*Root), folder: folder}
}

func (m *Manager) GetUser(ctx context.Context, user string) (*Root, error) {
	m.mu.RLock()
	r, ok := m.data[user]
	if ok {
		m.mu.RUnlock()
		return r, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := LoadRoot(m.folder, user)
	if err != nil {
		return nil, err
	}
	return r, r.Mount(ctx)
}

func (m *Manager) InitUser(ctx context.Context, stage3 string, user string) (*Root, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, err := CreateRoot(ctx, stage3, m.folder, user)
	if err != nil {
		return nil, err
	}
	err = r.Mount(ctx)
	if err != nil {
		return nil, err
	}
	return r, r.Update(ctx)
}

func (m *Manager) Close(ctx context.Context) error {
	var errs []error
	for _, r := range m.data {
		err := r.Close(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	var out error
	for _, err := range errs {
		if out != nil {
			out = errors.Join(out, err)
		} else {
			out = err
		}
	}
	return out
}
