# --- Переменные окружения для Docker Compose ---
export DB_HOST = localhost
export DB_PORT = 5432
export DB_USER = order_user
export DB_NAME = orders_db

# --- Название топика Kafka ---
KAFKA_TOPIC = orders

# --- Пути к скриптам ---
INIT_ROLE_SCRIPT = scripts/db/init_role.sql
INIT_TABLES_SCRIPT = scripts/db/init_tables.sql

# --- Тк команды не относяться к файлсам
.PHONY: all up down logs build run clean help

# Команда по умолчанию
all: build

# Запустить все сервисы (Kafka, Postgres) в фоновом режиме
up:
	@echo "--> Starting all services via Docker Compose..."
	docker-compose up -d

# Остановить и удалить все сервисы (эта команда не удаляет тома == Docker Volumes)
down:
	@echo "--> Stopping all services..."
	docker-compose down

# Показать логи всех сервисов
logs:
	@echo "--> Tailing logs..."
	docker-compose logs -f

# Инициализировать базу данных (после запуска 'up') ЭТО НЕ РАБОТАЕТ
db-init:
	@echo "--> Waiting for PostgreSQL to be ready..."
	@sleep 5
	@echo "Initializing database..."
	psql -h $(DB_HOST) -p $(DB_PORT) -U postgres -c "SELECT 'CREATE DATABASE $(DB_NAME) OWNER $(DB_USER)' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$(DB_NAME)')\gexec"
	psql -h $(DB_HOST) -p $(DB_PORT) -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE $(DB_NAME) TO $(DB_USER);"
	@PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f $(INIT_TABLES_SCRIPT)
	@echo "--> Database initialized successfully!"

# Создать топик в Kafka
kafka-create-topic:
	@echo "--> Creating Kafka topic: $(KAFKA_TOPIC)..."
	docker-compose exec kafka kafka-topics.sh --create --topic $(KAFKA_TOPIC) --bootstrap-server localhost:9092

# --- Сборка и запуск приложения ---

# Собрать Go приложение
build:
	go build -o bin/main cmd/main.go

# Запустить Go приложение (после сборки)
run: build
	./bin/main

# Очистить артефакты сборки
clean:
	rm -rf bin/

# Выводит команды с пояснениями
help:
	@echo "Available targets:"
	@echo "  up                  - Start Kafka and PostgreSQL via Docker"
	@echo "  down                - Stop all services"
	@echo "  logs                - View service logs"
	@echo "  db-init             - Initialize the database (run after 'up')"
	@echo "  kafka-create-topic  - Create the default Kafka topic"
	@echo "  build               - Build the Go application"
	@echo "  run                 - Run the Go application"
	@echo "  clean               - Clean build artifacts"