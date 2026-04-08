package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	recurrenceJSON, err := marshalRecurrenceParams(task.RecurrenceParams)
	if err != nil {
		return nil, err
	}

	const query = `
		INSERT INTO tasks (
			title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		task.Title,
		task.Description,
		task.Status,
		task.ParentID,
		task.ScheduledAt,
		task.RecurrenceType,
		recurrenceJSON,
		task.CreatedAt,
		task.UpdatedAt,
	)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) CreateRecurring(
	ctx context.Context,
	parent *taskdomain.Task,
	children []*taskdomain.Task,
) ([]taskdomain.Task, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	recurrenceJSON, err := marshalRecurrenceParams(parent.RecurrenceParams)
	if err != nil {
		return nil, err
	}

	const insertQuery = `
		INSERT INTO tasks (
			title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
	`

	parentRow := tx.QueryRow(
		ctx,
		insertQuery,
		parent.Title,
		parent.Description,
		parent.Status,
		parent.ParentID,
		parent.ScheduledAt,
		parent.RecurrenceType,
		recurrenceJSON,
		parent.CreatedAt,
		parent.UpdatedAt,
	)
	createdParent, err := scanTask(parentRow)
	if err != nil {
		return nil, err
	}

	createdChildren := make([]taskdomain.Task, 0, len(children))
	for _, child := range children {
		child.ParentID = &createdParent.ID
		row := tx.QueryRow(
			ctx,
			insertQuery,
			child.Title,
			child.Description,
			child.Status,
			child.ParentID,
			child.ScheduledAt,
			nil,
			nil,
			child.CreatedAt,
			child.UpdatedAt,
		)
		createdChild, scanErr := scanTask(row)
		if scanErr != nil {
			return nil, scanErr
		}
		createdChildren = append(createdChildren, *createdChild)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	allTasks := make([]taskdomain.Task, 0, 1+len(createdChildren))
	allTasks = append(allTasks, *createdParent)
	allTasks = append(allTasks, createdChildren...)
	return allTasks, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			updated_at = $4
		WHERE id = $5
		RETURNING id, title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) GetSeries(ctx context.Context, rootID int64) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
		FROM tasks
		WHERE id = $1 OR parent_id = $1
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, query, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, taskdomain.ErrNotFound
	}
	return tasks, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) DeleteSeries(ctx context.Context, rootID int64) error {
	const query = `DELETE FROM tasks WHERE id = $1 OR parent_id = $1`

	result, err := r.pool.Exec(ctx, query, rootID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, parent_id, scheduled_at, recurrence_type, recurrence_params, created_at, updated_at
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task           taskdomain.Task
		status         string
		parentID       *int64
		recurrenceType *string
		recurrenceRaw  []byte
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&parentID,
		&task.ScheduledAt,
		&recurrenceType,
		&recurrenceRaw,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)
	task.ParentID = parentID
	task.RecurrenceType = recurrenceType
	if len(recurrenceRaw) > 0 {
		var params taskdomain.RecurrenceParams
		if err := json.Unmarshal(recurrenceRaw, &params); err != nil {
			return nil, err
		}
		task.RecurrenceParams = &params
	}

	return &task, nil
}

func marshalRecurrenceParams(params *taskdomain.RecurrenceParams) ([]byte, error) {
	if params == nil {
		return nil, nil
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
