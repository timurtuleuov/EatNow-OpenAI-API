package db

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Драйвер для PG
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Драйвер для чтения файлов с диска
)

func RunMigrations(databaseURL string) {
	// 1. Получаем абсолютный путь к рабочей директории (корню проекта, откуда пишется go run .)
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ Не удалось получить рабочую директорию: %v", err)
	}

	// 2. Намертво сшиваем корень проекта и нужный путь к папке с миграциями
	// Получится что-то вроде: /Users/timur/project/db/migrations
	absolutePath := filepath.Join(wd, "db", "migrations")

	// 3. Формируем строку источника для golang-migrate (добавляем префикс file://)
	// Важно: filepath.ToSlash нужен для того, чтобы на Windows пути не ломались из-за обратных слэшей \
	migrationsSource := fmt.Sprintf("file://%s", filepath.ToSlash(absolutePath))

	log.Printf("📂 Ищу файлы миграций по пути: %s", absolutePath)

	// 4. Инициализируем мигратор
	m, err := migrate.New(migrationsSource, databaseURL)
	if err != nil {
		log.Fatalf("❌ Ошибка создания мигратора: %v", err)
	}

	// 5. Запускаем накат схемы
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("✅ Схема базы данных актуальна. Новых миграций нет.")
			return
		}
		log.Fatalf("❌ Ошибка выполнения миграций: %v", err)
	}

	log.Println("🚀 Миграции успешно применены!")
}
