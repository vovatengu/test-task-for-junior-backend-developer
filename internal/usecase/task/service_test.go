package task

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type testRepo struct {
	createFn          func(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	createRecurringFn func(ctx context.Context, parent *taskdomain.Task, children []*taskdomain.Task) ([]taskdomain.Task, error)
	getByIDFn         func(ctx context.Context, id int64) (*taskdomain.Task, error)
	deleteFn          func(ctx context.Context, id int64) error
}

func (r *testRepo) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	return r.createFn(ctx, task)
}

func (r *testRepo) CreateRecurring(ctx context.Context, parent *taskdomain.Task, children []*taskdomain.Task) ([]taskdomain.Task, error) {
	return r.createRecurringFn(ctx, parent, children)
}

func (r *testRepo) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if r.getByIDFn != nil {
		return r.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (r *testRepo) GetSeries(context.Context, int64) ([]taskdomain.Task, error) {
	return nil, nil
}
func (r *testRepo) Update(context.Context, *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}
func (r *testRepo) Delete(ctx context.Context, id int64) error {
	if r.deleteFn != nil {
		return r.deleteFn(ctx, id)
	}
	return nil
}
func (r *testRepo) DeleteSeries(context.Context, int64) error {
	return nil
}
func (r *testRepo) List(context.Context) ([]taskdomain.Task, error) { return nil, nil }

func TestServiceCreateOneTime(t *testing.T) {
	repo := &testRepo{
		createFn: func(_ context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
			task.ID = 10
			return task, nil
		},
		createRecurringFn: func(context.Context, *taskdomain.Task, []*taskdomain.Task) ([]taskdomain.Task, error) {
			t.Fatalf("unexpected CreateRecurring call")
			return nil, nil
		},
	}

	svc := NewService(repo)
	svc.now = func() time.Time { return time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC) }

	got, err := svc.Create(context.Background(), CreateInput{
		Title:       "Call patients",
		Description: "Morning checklist",
		Status:      taskdomain.StatusNew,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(got))
	}
	if got[0].ID != 10 {
		t.Fatalf("unexpected id: %d", got[0].ID)
	}
}

func TestServiceCreateRecurring(t *testing.T) {
	repo := &testRepo{
		createFn: func(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
			t.Fatalf("unexpected Create call")
			return nil, nil
		},
		createRecurringFn: func(_ context.Context, parent *taskdomain.Task, children []*taskdomain.Task) ([]taskdomain.Task, error) {
			if parent.RecurrenceType == nil || *parent.RecurrenceType != taskdomain.RecurrenceDaily {
				t.Fatalf("unexpected parent recurrence type")
			}
			if len(children) != 3 {
				t.Fatalf("expected 3 children, got %d", len(children))
			}
			parent.ID = 100
			out := make([]taskdomain.Task, 0, 1+len(children))
			out = append(out, *parent)
			for i := range children {
				children[i].ID = int64(i + 1)
				out = append(out, *children[i])
			}
			return out, nil
		},
	}

	svc := NewService(repo)
	daily := taskdomain.RecurrenceDaily
	created, err := svc.Create(context.Background(), CreateInput{
		Title:  "Daily round",
		Status: taskdomain.StatusInProgress,
		Recurrence: &RecurrenceInput{
			Type:      daily,
			StartDate: "2026-04-01",
			EndDate:   "2026-04-05",
			Daily: &DailyRecurrenceInput{
				Interval: 2,
			},
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if len(created) != 4 {
		t.Fatalf("expected 4 tasks (head + children), got %d", len(created))
	}
}

func TestCalculateRecurrenceDatesMonthly(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	got, err := calculateRecurrenceDates(start, end, taskdomain.RecurrenceMonthly, taskdomain.RecurrenceParams{
		DaysOfMonth: []int{5, 20},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(got))
	}
}

func TestValidateCreateInputInvalidEvenOdd(t *testing.T) {
	rType := taskdomain.RecurrenceEvenOdd
	_, err := validateCreateInput(CreateInput{
		Title:  "Task",
		Status: taskdomain.StatusNew,
		Recurrence: &RecurrenceInput{
			Type:      rType,
			StartDate: "2026-04-01",
			EndDate:   "2026-04-10",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceDeleteRejectsHeadTask(t *testing.T) {
	daily := taskdomain.RecurrenceDaily
	var deleteCalled bool
	repo := &testRepo{
		getByIDFn: func(_ context.Context, id int64) (*taskdomain.Task, error) {
			return &taskdomain.Task{ID: id, RecurrenceType: &daily}, nil
		},
		deleteFn: func(context.Context, int64) error {
			deleteCalled = true
			return nil
		},
	}
	svc := NewService(repo)
	err := svc.Delete(context.Background(), 42)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if deleteCalled {
		t.Fatal("Delete must not be called for head task")
	}
}

func TestServiceDeleteAllowsNonHeadTask(t *testing.T) {
	var deletedID int64
	repo := &testRepo{
		getByIDFn: func(_ context.Context, id int64) (*taskdomain.Task, error) {
			parent := int64(100)
			return &taskdomain.Task{ID: id, ParentID: &parent}, nil
		},
		deleteFn: func(_ context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	svc := NewService(repo)
	if err := svc.Delete(context.Background(), 7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 7 {
		t.Fatalf("expected delete id 7, got %d", deletedID)
	}
}

func TestServiceDeleteAllowsStandaloneTask(t *testing.T) {
	var deletedID int64
	repo := &testRepo{
		getByIDFn: func(_ context.Context, id int64) (*taskdomain.Task, error) {
			return &taskdomain.Task{ID: id}, nil
		},
		deleteFn: func(_ context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	svc := NewService(repo)
	if err := svc.Delete(context.Background(), 99); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != 99 {
		t.Fatalf("expected delete id 99, got %d", deletedID)
	}
}
