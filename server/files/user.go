package files

import (
	"context"
	"errors"
	"sync"
)

type Manager struct {
	mu     sync.Mutex
	data   map[string]*Root
	folder string
	stage3 string
}

func NewManager(folder, stage3 string) *Manager {
	return &Manager{data: make(map[string]*Root), folder: folder}
}

func (m *Manager) GetUser(ctx context.Context, user string) (*Root, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.data[user]
	if ok {
		return r, nil
	}
	r, err := CreateRoot(ctx, m.stage3, m.folder, user)
	if err != nil {
		return nil, err
	}
	err = r.Mount(ctx)
	if err != nil {
		return nil, err
	}
	m.data[user] = r
	return r, nil
}

func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var errs []error
	for _, r := range m.data {
		err := r.Close(ctx)
		if err != nil {
			errs = append(errs, err)
		}
	}
	m.data = nil
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
