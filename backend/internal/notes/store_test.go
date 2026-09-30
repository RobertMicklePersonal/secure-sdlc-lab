package notes

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCRUD(t *testing.T) {
	s := NewStore(10)

	n, err := s.Create("title", "body")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(n.ID) != 32 {
		t.Errorf("ID length = %d, want 32", len(n.ID))
	}

	got, err := s.Get(n.ID)
	if err != nil || got != n {
		t.Fatalf("Get = %+v, %v; want %+v", got, err, n)
	}

	u, err := s.Update(n.ID, "new", "changed")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if u.Title != "new" || u.Body != "changed" || !u.CreatedAt.Equal(n.CreatedAt) {
		t.Errorf("Update = %+v", u)
	}

	if err := s.Delete(n.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(n.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete: err = %v, want ErrNotFound", err)
	}
}

func TestMissing(t *testing.T) {
	s := NewStore(10)
	if _, err := s.Update("nope", "a", "b"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update: err = %v, want ErrNotFound", err)
	}
	if err := s.Delete("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete: err = %v, want ErrNotFound", err)
	}
}

func TestCapacity(t *testing.T) {
	s := NewStore(2)
	for i := 0; i < 2; i++ {
		if _, err := s.Create("t", ""); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	if _, err := s.Create("t", ""); !errors.Is(err, ErrFull) {
		t.Errorf("Create over capacity: err = %v, want ErrFull", err)
	}
}

func TestListNewestFirst(t *testing.T) {
	s := NewStore(10)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	i := 0
	s.now = func() time.Time { i++; return base.Add(time.Duration(i) * time.Minute) }

	first, _ := s.Create("first", "")
	second, _ := s.Create("second", "")

	list := s.List()
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Errorf("List order = %+v", list)
	}
}

func TestConcurrentCreate(t *testing.T) {
	s := NewStore(100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Create("t", "b"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := len(s.List()); got != 50 {
		t.Errorf("len(List) = %d, want 50", got)
	}
}
