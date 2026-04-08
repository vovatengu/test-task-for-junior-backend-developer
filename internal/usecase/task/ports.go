package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	CreateRecurring(ctx context.Context, parent *taskdomain.Task, children []*taskdomain.Task) ([]taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	GetSeries(ctx context.Context, rootID int64) ([]taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	DeleteSeries(ctx context.Context, rootID int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) ([]taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	GetSeries(ctx context.Context, rootID int64) ([]taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	DeleteSeries(ctx context.Context, rootID int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	ScheduledAt *time.Time
	Recurrence  *RecurrenceInput
}

type RecurrenceInput struct {
	Type      string
	StartDate string
	EndDate   string
	Daily     *DailyRecurrenceInput
	Monthly   *MonthlyRecurrenceInput
	Specific  *SpecificDatesRecurrenceInput
	EvenOdd   *EvenOddRecurrenceInput
}

type DailyRecurrenceInput struct {
	Interval int
}

type MonthlyRecurrenceInput struct {
	DaysOfMonth []int
}

type SpecificDatesRecurrenceInput struct {
	Dates []string
}

type EvenOddRecurrenceInput struct {
	IsEven *bool
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
}
