# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файл из `migrations/0001_create_tasks.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}/series`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}/series`

### Эндпоинты и параметры (кратко)

- `POST /api/v1/tasks` — создать одну задачу или серию задач.
  - Тело: `title`, `description`, `status`.
  - Для разовой задачи: `scheduled_at` (RFC3339, UTC).
  - Для серии: `recurrence` с полями `type`, `start_date`, `end_date` и параметрами типа:
    - `daily` -> `interval`
    - `monthly` -> `days_of_month`
    - `specific_dates` -> `dates`
    - `even_odd` -> `is_even`
  - Ответ: массив созданных задач (1 элемент для разовой, N элементов для серии).
- `GET /api/v1/tasks` — список всех задач.
  - Параметры: нет.
  - Ответ: массив задач.
- `GET /api/v1/tasks/{id}` — получить задачу по идентификатору.
  - Path-параметр: `id` (`int64`, `> 0`).
  - Ответ: массив из одного объекта задачи.
- `PUT /api/v1/tasks/{id}` — обновить поля задачи.
  - Path-параметр: `id`.
  - Тело: `title`, `description`, `status`.
  - Ответ: обновленный объект задачи.
- `DELETE /api/v1/tasks/{id}` — удалить одну задачу.
  - Path-параметр: `id`.
  - Ответ: `204 No Content`.
- `GET /api/v1/tasks/{id}/series` — получить всю серию по задаче серии.
  - Path-параметр: `id` (только `id` родительской/head-задачи серии).
  - Ответ: массив задач одной серии.
- `DELETE /api/v1/tasks/{id}/series` — удалить всю серию задач.
  - Path-параметр: `id` (только `id` родительской/head-задачи серии).
  - Ответ: `204 No Content`.

## Периодические задачи

### Поддерживаемые типы периодичности

- `daily`: каждые `n` дней
- `monthly`: по заданным дням месяца `1..30`
- `specific_dates`: только указанные даты
- `even_odd`: только четные или нечетные дни

### Принятые допущения

- Для разовой задачи периодичность не указывается (`recurrence_type = null`).
- Для периодической задачи обязательны `start_date` и `end_date` в формате `YYYY-MM-DD`.
- Все даты/время в БД хранятся как `TIMESTAMPTZ` в UTC.

### Примеры запросов

Разовая задача:

```json
{
  "title": "Связаться с клиентом",
  "description": "Подтвердить время приема",
  "status": "new",
  "scheduled_at": "2026-04-07T10:00:00Z"
}
```

Периодическая `daily`:

```json
{
  "title": "Обзвон пациентов",
  "description": "Утренний обзвон",
  "status": "in_progress",
  "recurrence": {
    "type": "daily",
    "start_date": "2026-04-01",
    "end_date": "2026-04-10",
    "interval": 2
  }
}
```

Периодическая `monthly`:

```json
{
  "title": "Инвентаризация",
  "description": "Плановая проверка",
  "status": "new",
  "recurrence": {
    "type": "monthly",
    "start_date": "2026-04-01",
    "end_date": "2026-06-30",
    "days_of_month": [5, 20]
  }
}
```
