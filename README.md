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
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

## Периодические задачи

### Поддерживаемые типы периодичности

- `daily`: каждые `n` дней (`recurrence_params.step`)
- `monthly`: по заданным дням месяца `1..30` (`recurrence_params.days_of_month`)
- `specific_dates`: только указанные даты (`recurrence_params.dates`)
- `even_odd`: только четные или нечетные дни (`recurrence_params.is_even`)

### Принятые допущения

- Для разовой задачи периодичность не указывается (`recurrence_type = null`).
- Для периодической задачи обязательны `start_date` и `end_date` в формате `YYYY-MM-DD`.
- Все даты/время в БД хранятся как `TIMESTAMPTZ` в UTC.
- Шаблон периодической задачи (head) имеет обычный статус (`new/in_progress/done/cancel`) и используется для связи серии по `parent_id`.
- `POST /tasks` возвращает массив созданных задач (для разовой задачи массив из одного элемента).

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
