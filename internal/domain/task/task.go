package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancel     Status = "cancel"
)

const (
	RecurrenceDaily         = "daily"
	RecurrenceMonthly       = "monthly"
	RecurrenceSpecificDates = "specific_dates"
	RecurrenceEvenOdd       = "even_odd"
)

type RecurrenceParams struct {
	Dates       []string `json:"dates,omitempty"`
	IsEven      *bool    `json:"is_even,omitempty"`
	DaysOfMonth []int    `json:"days_of_month,omitempty"`
	Step        int      `json:"step,omitempty"`
}

type Task struct {
	ID               int64             `json:"id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           Status            `json:"status"`
	ParentID         *int64            `json:"parent_id,omitempty"`
	ScheduledAt      time.Time         `json:"scheduled_at"`
	RecurrenceType   *string           `json:"recurrence_type,omitempty"`
	RecurrenceParams *RecurrenceParams `json:"recurrence_params,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone, StatusCancel:
		return true
	default:
		return false
	}
}
