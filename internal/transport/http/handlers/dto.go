package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	Recurrence  *recurrenceDTO    `json:"recurrence,omitempty"`
}

type recurrenceDTO struct {
	Type        string   `json:"type"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	Interval    int      `json:"interval,omitempty"`
	DaysOfMonth []int    `json:"days_of_month,omitempty"`
	Dates       []string `json:"dates,omitempty"`
	IsEven      *bool    `json:"is_even,omitempty"`
}

type taskDTO struct {
	ID               int64                        `json:"id"`
	Title            string                       `json:"title"`
	Description      string                       `json:"description"`
	Status           taskdomain.Status            `json:"status"`
	ParentID         *int64                       `json:"parent_id,omitempty"`
	ScheduledAt      time.Time                    `json:"scheduled_at"`
	RecurrenceType   *string                      `json:"recurrence_type,omitempty"`
	RecurrenceParams *taskdomain.RecurrenceParams `json:"recurrence_params,omitempty"`
	Recurrence       *recurrenceDTO               `json:"recurrence,omitempty"`
	CreatedAt        time.Time                    `json:"created_at"`
	UpdatedAt        time.Time                    `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	recurrence := toRecurrenceDTO(task.RecurrenceType, task.RecurrenceParams, task.ScheduledAt)
	return taskDTO{
		ID:               task.ID,
		Title:            task.Title,
		Description:      task.Description,
		Status:           task.Status,
		ParentID:         task.ParentID,
		ScheduledAt:      task.ScheduledAt,
		RecurrenceType:   task.RecurrenceType,
		RecurrenceParams: task.RecurrenceParams,
		Recurrence:       recurrence,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
	}
}

func toRecurrenceDTO(rType *string, params *taskdomain.RecurrenceParams, scheduledAt time.Time) *recurrenceDTO {
	if rType == nil {
		return nil
	}
	dto := &recurrenceDTO{
		Type:      *rType,
		StartDate: scheduledAt.UTC().Format("2006-01-02"),
		EndDate:   scheduledAt.UTC().Format("2006-01-02"),
	}
	if params == nil {
		return dto
	}
	dto.Interval = params.Step
	dto.DaysOfMonth = params.DaysOfMonth
	dto.Dates = params.Dates
	dto.IsEven = params.IsEven
	return dto
}
