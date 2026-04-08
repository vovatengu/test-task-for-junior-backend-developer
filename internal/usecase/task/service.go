package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) ([]taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	if normalized.Recurrence == nil {
		scheduledAt := now
		if normalized.ScheduledAt != nil {
			scheduledAt = normalized.ScheduledAt.UTC()
		}
		model := &taskdomain.Task{
			Title:       normalized.Title,
			Description: normalized.Description,
			Status:      normalized.Status,
			ScheduledAt: scheduledAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		created, createErr := s.repo.Create(ctx, model)
		if createErr != nil {
			return nil, createErr
		}

		return []taskdomain.Task{*created}, nil
	}

	start, end, err := parseRange(normalized.Recurrence.StartDate, normalized.Recurrence.EndDate)
	if err != nil {
		return nil, err
	}
	params := recurrenceParamsFromInput(*normalized.Recurrence)
	dates, err := calculateRecurrenceDates(start, end, normalized.Recurrence.Type, params)
	if err != nil {
		return nil, err
	}
	if len(dates) == 0 {
		return nil, fmt.Errorf("%w: recurrence produced no dates", ErrInvalidInput)
	}

	parent := &taskdomain.Task{
		Title:            normalized.Title,
		Description:      normalized.Description,
		Status:           normalized.Status,
		ScheduledAt:      start,
		RecurrenceType:   &normalized.Recurrence.Type,
		RecurrenceParams: &params,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	children := make([]*taskdomain.Task, 0, len(dates))
	for _, date := range dates {
		children = append(children, &taskdomain.Task{
			Title:       normalized.Title,
			Description: normalized.Description,
			Status:      normalized.Status,
			ScheduledAt: date,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return s.repo.CreateRecurring(ctx, parent, children)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) GetSeries(ctx context.Context, rootID int64) ([]taskdomain.Task, error) {
	if rootID <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.GetSeries(ctx, rootID)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) DeleteSeries(ctx context.Context, rootID int64) error {
	if rootID <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.DeleteSeries(ctx, rootID)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	if input.Recurrence == nil {
		return input, nil
	}
	input.Recurrence.Type = strings.TrimSpace(input.Recurrence.Type)
	input.Recurrence.StartDate = strings.TrimSpace(input.Recurrence.StartDate)
	input.Recurrence.EndDate = strings.TrimSpace(input.Recurrence.EndDate)
	if input.Recurrence.Type == "" {
		return CreateInput{}, fmt.Errorf("%w: recurrence.type is required", ErrInvalidInput)
	}
	if input.Recurrence.StartDate == "" || input.Recurrence.EndDate == "" {
		return CreateInput{}, fmt.Errorf("%w: start_date and end_date are required for recurring tasks", ErrInvalidInput)
	}
	if err := validateRecurrenceShape(*input.Recurrence); err != nil {
		return CreateInput{}, err
	}
	params := recurrenceParamsFromInput(*input.Recurrence)
	if err := validateRecurrenceParams(input.Recurrence.Type, params); err != nil {
		return CreateInput{}, err
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateRecurrenceParams(rType string, params taskdomain.RecurrenceParams) error {
	switch rType {
	case taskdomain.RecurrenceDaily:
		if params.Step <= 0 {
			return fmt.Errorf("%w: step must be greater than 0", ErrInvalidInput)
		}
	case taskdomain.RecurrenceMonthly:
		if len(params.DaysOfMonth) == 0 {
			return fmt.Errorf("%w: days_of_month is required", ErrInvalidInput)
		}
		for _, day := range params.DaysOfMonth {
			if day < 1 || day > 30 {
				return fmt.Errorf("%w: days_of_month must contain values from 1 to 30", ErrInvalidInput)
			}
		}
	case taskdomain.RecurrenceSpecificDates:
		if len(params.Dates) == 0 {
			return fmt.Errorf("%w: dates is required", ErrInvalidInput)
		}
	case taskdomain.RecurrenceEvenOdd:
		if params.IsEven == nil {
			return fmt.Errorf("%w: is_even is required", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown recurrence_type", ErrInvalidInput)
	}
	return nil
}

func recurrenceParamsFromInput(in RecurrenceInput) taskdomain.RecurrenceParams {
	params := taskdomain.RecurrenceParams{}
	if in.Daily != nil {
		params.Step = in.Daily.Interval
	}
	if in.Monthly != nil {
		params.DaysOfMonth = in.Monthly.DaysOfMonth
	}
	if in.Specific != nil {
		params.Dates = in.Specific.Dates
	}
	if in.EvenOdd != nil {
		params.IsEven = in.EvenOdd.IsEven
	}
	return params
}

func validateRecurrenceShape(in RecurrenceInput) error {
	switch in.Type {
	case taskdomain.RecurrenceDaily:
		if in.Daily == nil {
			return fmt.Errorf("%w: recurrence.daily is required for type daily", ErrInvalidInput)
		}
	case taskdomain.RecurrenceMonthly:
		if in.Monthly == nil {
			return fmt.Errorf("%w: recurrence.monthly is required for type monthly", ErrInvalidInput)
		}
	case taskdomain.RecurrenceSpecificDates:
		if in.Specific == nil {
			return fmt.Errorf("%w: recurrence.specific is required for type specific_dates", ErrInvalidInput)
		}
	case taskdomain.RecurrenceEvenOdd:
		if in.EvenOdd == nil {
			return fmt.Errorf("%w: recurrence.even_odd is required for type even_odd", ErrInvalidInput)
		}
	}
	return nil
}

func parseRange(startDate, endDate string) (time.Time, time.Time, error) {
	const layout = "2006-01-02"
	start, err := time.Parse(layout, startDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid start_date format, expected YYYY-MM-DD", ErrInvalidInput)
	}
	end, err := time.Parse(layout, endDate)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid end_date format, expected YYYY-MM-DD", ErrInvalidInput)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: end_date must be greater than or equal to start_date", ErrInvalidInput)
	}
	return start.UTC(), end.UTC(), nil
}

func calculateRecurrenceDates(start, end time.Time, recurrenceType string, params taskdomain.RecurrenceParams) ([]time.Time, error) {
	dates := make([]time.Time, 0)
	switch recurrenceType {
	case taskdomain.RecurrenceDaily:
		for d := start; !d.After(end); d = d.AddDate(0, 0, params.Step) {
			dates = append(dates, d.UTC())
		}
	case taskdomain.RecurrenceMonthly:
		allowedDays := make(map[int]struct{}, len(params.DaysOfMonth))
		for _, day := range params.DaysOfMonth {
			allowedDays[day] = struct{}{}
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			if _, ok := allowedDays[d.Day()]; ok {
				dates = append(dates, d.UTC())
			}
		}
	case taskdomain.RecurrenceSpecificDates:
		for _, rawDate := range params.Dates {
			parsed, err := time.Parse("2006-01-02", strings.TrimSpace(rawDate))
			if err != nil {
				return nil, fmt.Errorf("%w: invalid date %q in recurrence_params.dates", ErrInvalidInput, rawDate)
			}
			parsed = parsed.UTC()
			if (parsed.Equal(start) || parsed.After(start)) && (parsed.Equal(end) || parsed.Before(end)) {
				dates = append(dates, parsed)
			}
		}
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	case taskdomain.RecurrenceEvenOdd:
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			dayIsEven := d.Day()%2 == 0
			if *params.IsEven == dayIsEven {
				dates = append(dates, d.UTC())
			}
		}
	default:
		return nil, fmt.Errorf("%w: unknown recurrence_type", ErrInvalidInput)
	}
	return dates, nil
}
