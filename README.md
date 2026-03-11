# CrackHash - Distributed Hash Cracking System

Распределённая отказоустойчивая система для взлома MD5 хэшей методом brute-force перебора.

## Описание

Система CrackHash предназначена для взлома MD5 хэшей через перебор словаря, сгенерированного на основе алфавита.

1. Клиент отправляет менеджеру MD5-хэш и максимальную длину искомого слова
2. Менеджер разбивает задачу на подзадачи, сохраняет в MongoDB и публикует в RabbitMQ
3. Воркеры получают подзадачи из очереди, выполняют перебор и отправляют прогресс обратно через RabbitMQ
4. Менеджер агрегирует прогресс из MongoDB и предоставляет клиенту результат

## Архитектура системы

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                  CLIENT                                      │
│                                                                              │
│    POST /api/hash/crack         GET /api/hash/status?requestId=<UUID>        │
│    {hash, maxLength}            → {status, progress, data}                   │
└───────────────────────────────────┬──────────────────────────────────────────┘
                                    │ External API (JSON)
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                           MANAGER SERVICE                                    │
│                              (Port 8080)                                     │
│                                                                              │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────────────────┐ │
│  │ Task Management  │  │ SubTask Sender   │  │ Progress Aggregation       │ │
│  │                  │  │ (background)     │  │                            │ │
│  │ • Split ranges   │  │ • FindPending()  │  │ • Consume tasks_progress   │ │
│  │ • Save to MongoDB│  │ • Publish to MQ  │  │ • Consume dead_letter      │ │
│  │ • Return taskID  │  │ • MarkSent()     │  │ • Upsert to MongoDB       │ │
│  └──────────────────┘  └──────────────────┘  └────────────────────────────┘ │
└────────┬────────────────────────┬──────────────────────────┬────────────────┘
         │                        │                          │
         │ MongoDB                │ RabbitMQ                 │ RabbitMQ
         │ (w=majority)           │ "tasks" queue            │ "tasks_progress"
         ▼                        ▼                          │ "dead_letter"
┌─────────────────┐    ┌──────────────────┐                  │
│  MongoDB RS     │    │    RabbitMQ      │                  │
│  (rs0)          │    │                  │◄─────────────────┘
│                 │    │  tasks (quorum)  │
│  mongo1 (P)     │    │  tasks_progress  │
│  mongo2 (S)     │    │  dead_letter     │
│  mongo3 (S)     │    │                  │
└─────────────────┘    └───────┬──────────┘
                               │ "tasks" queue
                               │ (competing consumers)
                               ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          WORKER SERVICES (1..N)                              │
│                                                                              │
│   ┌─────────────────────┐   ┌─────────────────────┐                         │
│   │     Worker 1        │   │     Worker 2        │   ...                   │
│   │  Range: [0, N)      │   │  Range: [N, 2N)     │                         │
│   │                     │   │                     │                         │
│   │ • Consume from MQ   │   │ • Consume from MQ   │                         │
│   │ • MD5 brute-force   │   │ • MD5 brute-force   │                         │
│   │ • Publish progress  │   │ • Publish progress  │                         │
│   │ • ACK after done    │   │ • ACK after done    │                         │
│   └─────────────────────┘   └─────────────────────┘                         │
└──────────────────────────────────────────────────────────────────────────────┘
```

![Alt text](schema.jpg)

### Компоненты

| Компонент | Описание |
|-----------|----------|
| **Manager** | HTTP API для клиентов. Создаёт подзадачи, сохраняет в MongoDB, публикует в RabbitMQ. Потребляет прогресс и dead letters. |
| **Worker (x2+)** | Получает подзадачи из очереди "tasks" (competing consumers). Выполняет brute-force перебор. Отправляет прогресс в "tasks_progress". ACK после завершения. |
| **RabbitMQ** | Брокер сообщений. Три очереди: tasks (quorum), tasks_progress, dead_letter. |
| **MongoDB RS** | Replica Set (1 primary + 2 secondary). Хранит подзадачи (subtasks) и прогресс (task_progress). Write concern: majority. |

## Sequence Diagrams

### 1. Создание задачи

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Manager
    participant MongoDB
    participant RabbitMQ
    participant Worker1
    participant Worker2

    Client->>+Manager: POST /api/hash/crack {hash, maxLength}
    Note over Manager: Рассчитать SearchSpace, разбить на maxLength подзадач
    Manager->>MongoDB: CreateBatch(subtasks, status=pending)
    MongoDB-->>Manager: OK (w=majority)
    Manager-->>-Client: {requestId: UUID}

    Note over Manager: Фоновая горутина (каждые retry_send_period)
    Manager->>MongoDB: FindPending()
    MongoDB-->>Manager: [pending subtasks]
    Manager->>RabbitMQ: Publish to "tasks" (persistent)
    Manager->>MongoDB: MarkSent(subtask)

    RabbitMQ->>Worker1: Deliver subtask [0, N)
    RabbitMQ->>Worker2: Deliver subtask [N, 2N)

    loop Каждые notify_period
        Worker1->>RabbitMQ: Publish progress to "tasks_progress"
        Worker2->>RabbitMQ: Publish progress to "tasks_progress"
        RabbitMQ->>Manager: Deliver progress
        Manager->>MongoDB: Upsert progress
    end

    Worker1->>RabbitMQ: ACK (task done)
    Worker2->>RabbitMQ: ACK (task done)
```

### 2. Запрос статуса

```mermaid
sequenceDiagram
    autonumber
    participant Client
    participant Manager
    participant MongoDB

    Client->>Manager: GET /api/hash/status?requestId=UUID
    Manager->>MongoDB: Collect progress by task_id
    MongoDB-->>Manager: [worker progresses]
    Note over Manager: mergeProgress(): агрегация итераций, статусов, результатов
    Manager-->>Client: {status, progress, data}
```

### 3. Обработка стоп-слова "bom" и DLQ

```mermaid
sequenceDiagram
    autonumber
    participant RabbitMQ
    participant Worker
    participant Manager
    participant MongoDB

    RabbitMQ->>Worker: Deliver subtask
    Note over Worker: Нашёл слово "bom" → panic
    Note over Worker: Контейнер перезапускается (restart: on-failure)
    Note over RabbitMQ: Сообщение не ACK-нуто → requeue

    RabbitMQ->>Worker: Redelivery (попытка 2)
    Note over Worker: panic again

    RabbitMQ->>Worker: Redelivery (попытка 3)
    Note over Worker: panic again

    Note over RabbitMQ: x-delivery-limit (3) превышен
    RabbitMQ->>RabbitMQ: Move to "dead_letter" queue

    RabbitMQ->>Manager: Deliver dead letter
    Manager->>MongoDB: Upsert progress (status=ERROR)
    Note over Manager: Клиент увидит status=ERROR
```

## Отказоустойчивость

| Сценарий | Поведение |
|----------|-----------|
| **Падение менеджера** | Ответы воркеров сохраняются в RabbitMQ до рестарта. MongoDB хранит все данные. |
| **Падение primary MongoDB** | Автоматическое переключение на secondary. Система продолжает работу. |
| **Падение RabbitMQ** | Подзадачи остаются в MongoDB со статусом "pending". Фоновая горутина отправит их после восстановления. Персистентные сообщения сохраняются при рестарте. |
| **Падение воркера** | Сообщение не ACK-нуто → RabbitMQ переотправляет другому воркеру. Docker перезапускает упавший контейнер. |
| **Стоп-слово "bom"** | Воркер паникует. После 3 попыток сообщение переходит в DLQ. Задача помечается ERROR. Другие воркеры продолжают работу. |

## API

### External API (Manager)

#### POST /api/hash/crack

Создание запроса на взлом хэша.

**Request:**
```json
{
  "hash": "e2fc714c4727ee9395f324cd2e7f331f",
  "maxLength": 4
}
```

**Response (200 OK):**
```json
{
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### GET /api/hash/status?requestId=UUID

Получение статуса выполнения запроса.

**Response (IN_PROGRESS):**
```json
{
  "status": "IN_PROGRESS",
  "progress": 65,
  "data": []
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
  "data": []
}
```

#### GET /health

Проверка работоспособности. Возвращает `OK`.

### Очереди RabbitMQ

| Очередь | Тип | Направление | Описание |
|---------|-----|-------------|----------|
| `tasks` | quorum | Manager → Workers | Подзадачи на обработку. x-delivery-limit: 3, DLX → dead_letter |
| `tasks_progress` | standard | Workers → Manager | Прогресс выполнения подзадач |
| `dead_letter` | standard | RabbitMQ → Manager | Подзадачи, превысившие лимит попыток |

Все сообщения персистентные (DeliveryMode: Persistent).

## Развёртывание

### Требования

- Docker + Docker Compose

### Запуск

```bash
cd docker
docker compose up --build
```

Для запуска с масштабированием воркеров:
```bash
docker compose up --build --scale worker-service=3
```

### Сервисы

| Сервис | Порт | Описание |
|--------|------|----------|
| manager-service | 8080 | HTTP API для клиентов |
| worker-service (x2) | - | Обработчики подзадач |
| rabbitmq | 5672, 15672 | Брокер сообщений (+ Management UI) |
| mongo1 | 27017 | MongoDB Primary |
| mongo2 | - | MongoDB Secondary |
| mongo3 | - | MongoDB Secondary |
| mongo-init | - | Инициализация Replica Set (одноразовый) |

### Порядок запуска

1. MongoDB ноды запускаются
2. `mongo-init` инициализирует Replica Set после healthcheck mongo1
3. RabbitMQ запускается и проходит healthcheck
4. Manager стартует после mongo-init и rabbitmq
5. Workers стартуют после manager

## Конфигурация

### Manager (configs/manager.yaml)

```yaml
http:
  port: 8080

storage:
  db: "tasks"
  url: mongodb://mongo1:27017,mongo2:27017,mongo3:27017/tasks_db?replicaSet=rs0&w=majority

broker:
  url: amqp://guest:guest@rabbitmq:5672/
  requeue_limit: 3

hash_cracker:
  alphabet: "abcdefghijklmnopqrstuvwxyz0123456789"

retry_send_period: 5s

logger:
  level: INFO
  is_json: false
```

### Worker (configs/worker.yaml)

```yaml
http:
  port: 8081

broker:
  url: amqp://guest:guest@rabbitmq:5672/
  requeue_limit: 3

workers:
  max_parallel: 1
  notify_period: 5s

logger:
  level: INFO
  is_json: false
```

## Пространство перебора

- **Алфавит:** конфигурируется (по умолчанию a-z, 0-9)
- **Длина строк:** от 1 до `maxLength` включительно
- **Размер пространства:** для алфавита размера N и максимальной длины L: N + N² + ... + Nᴸ
- **Количество подзадач:** равно `maxLength`

SearchSpace преобразует индекс в слово:
- Индекс 0 → "a", Индекс 35 → "9", Индекс 36 → "aa", Индекс 1331 → "99"

## Логирование

Структурированное логирование с помощью `slog/log`. Логи включают:

- Приём запросов от клиента
- Создание и отправку подзадач
- Получение прогресса от воркеров
- Обработку dead letters
- Ошибки взаимодействия с MongoDB и RabbitMQ

## OpenAPI документация

- Manager API: `docs/manager-api-openapi.yaml`
- Worker API: `docs/worker-api-openapi.yaml`

Для просмотра: [Swagger Editor](https://editor.swagger.io/)