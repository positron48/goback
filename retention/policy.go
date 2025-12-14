package retention

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"backup-tool/utils"
)

type RetentionPolicy struct {
	Daily   int
	Weekly  int
	Monthly int
	Yearly  int
}

type BackupFile struct {
	Path string
	Time time.Time
}

// ApplyRetention применяет политику хранения к бэкапам
func ApplyRetention(backupDir, subdirectory, backupName string, policy RetentionPolicy) error {
	backupPath := filepath.Join(backupDir, subdirectory)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil // Директория не существует, нечего чистить
	}

	// Получаем все файлы бэкапов, фильтруя по имени бэкапа
	files, err := getBackupFiles(backupPath, backupName)
	if err != nil {
		return fmt.Errorf("failed to get backup files: %w", err)
	}

	if len(files) == 0 {
		return nil
	}

	// Определяем файлы для сохранения
	toKeep := determineFilesToKeep(files, policy)

	// Удаляем файлы, которые не нужно сохранять
	for _, file := range files {
		shouldKeep := false
		for _, keepFile := range toKeep {
			if keepFile.Path == file.Path {
				shouldKeep = true
				break
			}
		}

		if !shouldKeep {
			if err := os.Remove(file.Path); err != nil {
				fmt.Printf("Warning: failed to remove old backup %s: %v\n", file.Path, err)
			} else {
				fmt.Printf("Removed old backup: %s\n", filepath.Base(file.Path))
			}
		}
	}

	return nil
}

func getBackupFiles(dir, backupName string) ([]BackupFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// Префикс для фильтрации файлов по имени бэкапа
	namePrefix := backupName + "-"

	var files []BackupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Фильтруем файлы по префиксу имени бэкапа
		entryName := entry.Name()
		// Убираем расширение для проверки префикса
		baseName := entryName
		if idx := strings.LastIndex(entryName, "."); idx != -1 {
			baseName = entryName[:idx]
		}

		// Проверяем, что имя файла начинается с {backupName}-
		if !strings.HasPrefix(baseName, namePrefix) {
			continue
		}

		path := filepath.Join(dir, entryName)
		t, err := utils.ParseDateFromFilename(entryName)
		if err != nil {
			// Пропускаем файлы, из которых нельзя извлечь дату
			continue
		}

		files = append(files, BackupFile{
			Path: path,
			Time: t,
		})
	}

	return files, nil
}

func determineFilesToKeep(files []BackupFile, policy RetentionPolicy) []BackupFile {
	if len(files) == 0 {
		return files
	}

	// Сортируем по времени (от старых к новым)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Time.Before(files[j].Time)
	})

	var toKeep []BackupFile

	// Группируем по периодам
	dailyAnchors := getAnchors(files, func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	})
	weeklyAnchors := getAnchors(files, func(t time.Time) time.Time {
		// Находим начало недели (понедельник)
		weekStart := t
		for weekStart.Weekday() != time.Monday {
			weekStart = weekStart.AddDate(0, 0, -1)
		}
		return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, t.Location())
	})
	monthlyAnchors := getAnchors(files, func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	})
	yearlyAnchors := getAnchors(files, func(t time.Time) time.Time {
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	})

	// Берем N последних якорных точек каждого типа
	toKeep = append(toKeep, getLastN(dailyAnchors, policy.Daily)...)
	toKeep = append(toKeep, getLastN(weeklyAnchors, policy.Weekly)...)
	toKeep = append(toKeep, getLastN(monthlyAnchors, policy.Monthly)...)
	toKeep = append(toKeep, getLastN(yearlyAnchors, policy.Yearly)...)

	// Убираем дубликаты
	unique := make(map[string]BackupFile)
	for _, file := range toKeep {
		unique[file.Path] = file
	}

	result := make([]BackupFile, 0, len(unique))
	for _, file := range unique {
		result = append(result, file)
	}

	return result
}

// getAnchors возвращает последний бэкап для каждого периода
func getAnchors(files []BackupFile, periodFunc func(time.Time) time.Time) []BackupFile {
	anchors := make(map[string]BackupFile)

	for _, file := range files {
		periodStart := periodFunc(file.Time)
		periodKey := periodStart.Format("2006-01-02")
		if existing, exists := anchors[periodKey]; !exists || file.Time.After(existing.Time) {
			anchors[periodKey] = file
		}
	}

	result := make([]BackupFile, 0, len(anchors))
	for _, file := range anchors {
		result = append(result, file)
	}

	return result
}

func getLastN(files []BackupFile, n int) []BackupFile {
	if n <= 0 {
		return nil
	}

	// Сортируем по времени (от старых к новым)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Time.Before(files[j].Time)
	})

	if len(files) <= n {
		return files
	}

	return files[len(files)-n:]
}

