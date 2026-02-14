# CrackHash - Distributed Hash Cracking System

Распределенная система для взлома MD5 хэшей методом brute-force перебора.

## Архитектура системы

Система состоит из двух типов сервисов:

### Manager (Менеджер)
- Принимает запросы от клиента на взлом хэша
- Управляет пулом воркеров (регистрация, health-check)
- Разбивает задачу на подзадачи и распределяет их между воркерами
- Агрегирует результаты от воркеров
- Предоставляет REST API для клиентов

### Worker (Воркер)
- Регистрируется в менеджере при старте
- Принимает задачи от менеджера
- Выполняет перебор слов в заданном диапазоне алфавита
- Вычисляет MD5 хэш для каждой строки
- Отправляет результаты обратно менеджеру
- Предоставляет Internal API для менеджера

## Схема взаимодействия компонентов

```
┌─────────┐
│ Client  │
└────┬────┘
     │ POST /api/hash/crack
     │ GET /api/hash/status?requestId=<UUID>
     │
┌────▼────────────────────────┐
│   Manager Service           │
│  - Task distribution        │
│  - Result aggregation       │
│  - Worker health checking   │
└─────┬───────────────────────┘
      │
      │ GET /api/hash/register-worker (worker registration)
      │ POST /api/v1/tasks/ (create task)
      │ PUT /api/v1/tasks/{id}/do (execute task)
      │ GET /api/v1/tasks/{id}/progress (get progress)
      │
      ▼
┌─────────────────────────────┐
│  Worker Services (1..N)     │
│  - Word generation          │
│  - MD5 hashing              │
│  - Result reporting         │
└─────────────────────────────┘
```

## Инструкция по запуску

### Требования
- Docker
- Docker Compose

## Описание API

### Client API (Manager)

#### POST /api/hash/crack
Создание запроса на взлом хэша.

**Request:**
```json
{
  "hash": "e2fc714c4727ee9395f324cd2e7f331f",
  "maxLength": 4
}
```

**Response:**
```json
{
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Status Codes:**
- `200 OK` - запрос принят
- `400 Bad Request` - неверный формат запроса
- `500 Internal Server Error` - внутренняя ошибка сервера

#### GET /api/hash/status?requestId=<UUID>
Получение статуса выполнения запроса.

**Parameters:**
- `requestId` - UUID запроса, полученный при создании

**Response (IN_PROGRESS):**
```json
{
  "status": "IN_PROGRESS",
  "progress": 65,
  "data": null
}
```

**Response (READY):**
```json
{
  "status": "READY",
  "progress": 100,
  "data": ["abcd"]
}
```

**Response (ERROR):**
```json
{
  "status": "ERROR",
  "progress": 50,
  "data": null
}
```

**Status Codes:**
- `200 OK` - успешный запрос
- `400 Bad Request` - неверный requestId
- `500 Internal Server Error` - внутренняя ошибка

**Возможные статусы:**
- `NOT_STARTED` - задача еще не началась
- `IN_PROGRESS` - задача выполняется
- `READY` - задача завершена успешно
- `ERROR` - произошла ошибка

### Internal API (Worker)

#### POST /api/v1/tasks/
Создание задачи на воркере.

**Request:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "target_hash": "e2fc714c4727ee9395f324cd2e7f331f",
  "iteration_alphabet": "abcdefgh",
  "max_length": 4
}
```

**Response:**
- `200 OK` - задача создана
- `400 Bad Request` - неверный формат
- `409 Conflict` - задача уже существует

#### PUT /api/v1/tasks/{task_id}/do
Запуск выполнения задачи.

**Response:**
- `200 OK` - задача запущена
- `404 Not Found` - задача не найдена

#### GET /api/v1/tasks/{task_id}/progress
Получение прогресса выполнения задачи.

**Response:**
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "IN_PROGRESS",
  "iterations_done": 2349,
  "total_iterations": 10000,
  "result": ["abcd"]
}
```

#### DELETE /api/v1/tasks/{task_id}
Удаление задачи.

**Response:**
- `200 OK` - задача удалена
- `404 Not Found` - задача не найдена

## Конфигурационные параметры

### Manager (configs/manager.yaml)

```yaml
http:
  port: 8080                    # Порт HTTP сервера

healthcheck:
  period: 10s                   # Период проверки здоровья воркеров
  max_tries: 5                  # Максимальное количество попыток

hash_cracker:
  alphabet: "abcdefghijklmnopqrstuvwxyz0123456789"  # Алфавит для перебора (a-z, 0-9)
```

### Worker (configs/worker.yaml)

```yaml
http:
  port: 8080                    # Порт HTTP сервера

registerer:
  register_url: "http://manager-service:8080/api/hash/register-worker"  # URL для регистрации

notifier:
  notify_url: "http://manager-service:8080/api/tasks/ready"  # URL для отправки результатов

workers:
  max_parallel: 10              # Максимальное количество параллельных операций
```

## Логирование

Система использует структурированное логирование с помощью `log/slog`. Логи включают:
- Прием запросов от клиента
- Регистрацию воркеров
- Распределение задач между воркерами
- Процесс выполнения задач
- Формирование итогового результата
- Ошибки и панические ситуации

Логи выводятся в stdout и могут быть просмотрены через:
```bash
docker-compose logs -f manager-service
docker-compose logs -f worker-service
```

## Архитектурные решения

1. **Распределение алфавита:** Менеджер делит алфавит на равные части по количеству воркеров, каждый воркер перебирает свою часть алфавита для всех длин от 1 до maxLength.

2. **Health Checking:** Менеджер периодически проверяет здоровье воркеров. При падении воркера, его задачи помечаются как ERROR.

3. **Хранение состояния:** Все данные о запросах и воркерах хранятся в памяти с использованием потокобезопасных структур (mutex).

4. **Регистрация воркеров:** Воркеры регистрируются в менеджере при старте, получая уникальный UUID.

5. **Итеративная генерация:** Воркеры не хранят все комбинации в памяти, а генерируют их итеративно.