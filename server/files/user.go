package files

import (
	"context"
	"errors"
	"iter"
	"os"
	"path"
	"sync"
)

type Manager struct {
	mu     sync.Mutex
	data   map[string]*Root
	folder string
}

func NewManager(folder string) *Manager {
	return &Manager{data: make(map[string]*Root), folder: folder}
}

func (m *Manager) GetUser(ctx context.Context, user string) (*Root, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.data[user]
	if ok {
		return r, nil
	}

	r, err := LoadRoot(m.folder, user)
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
	m.data[user] = r
	return r, r.Update(ctx)
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

func (m *Manager) InitUsers(ctx context.Context, users iter.Seq[string]) <-chan error {
	errc := make(chan error, 1)
	var wg sync.WaitGroup
	for user := range users {
		f, err := os.Open(path.Join(m.folder, user))
		if err == nil {
			_ = f.Close()
			continue
		}
		if !os.IsNotExist(err) {
			panic(err)
		}
		wg.Go(func() {
			_, err = m.InitUser(ctx, "", user)
			errc <- err
		})
	}
	res := make(chan error, 1)
	go func() {
		wg.Wait()
		close(errc)
	}()
	go func() {
		var err error
		for e := range errc {
			if e == nil {
				continue
			}
			if err == nil {
				err = e
			} else {
				err = errors.Join(err, e)
			}
		}
		res <- err
		close(res)
	}()
	return res
}
