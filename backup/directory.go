package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyDirectory копирует директорию с поддержкой exclude_patterns
func CopyDirectory(source, destination string, excludePatterns []string) error {
	// Создаем целевую директорию
	if err := os.MkdirAll(destination, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Нормализуем пути для корректной работы
	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for source: %w", err)
	}

	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for destination: %w", err)
	}

	return filepath.Walk(absSource, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Пропускаем файлы/директории, к которым нет доступа
			return nil
		}

		relPath, err := filepath.Rel(absSource, path)
		if err != nil {
			return err
		}

		// Пропускаем корневую директорию (relPath == ".")
		if relPath == "." {
			return nil
		}

		// Проверяем exclude patterns
		if shouldExclude(relPath, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := filepath.Join(absDestination, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Проверяем, является ли это симлинком
		// filepath.Walk использует os.Lstat, поэтому info содержит информацию о симлинке, а не о цели
		if info.Mode()&os.ModeSymlink != 0 {
			// Копируем симлинк как симлинк
			target, err := os.Readlink(path)
			if err != nil {
				// Не удалось прочитать симлинк, пропускаем
				return nil
			}
			// Проверяем, существует ли уже файл/симлинк по целевому пути
			if _, err := os.Lstat(destPath); err == nil {
				// Файл уже существует, удаляем его
				if err := os.Remove(destPath); err != nil {
					return fmt.Errorf("failed to remove existing file for symlink: %w", err)
				}
			}
			// Создаем новый симлинк
			return os.Symlink(target, destPath)
		}

		// Обычный файл - проверяем, что он все еще существует перед копированием
		// (может быть удален между моментом обнаружения и копированием)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			// Файл не существует, пропускаем
			return nil
		}

		return copyFile(path, destPath, info.Mode())
	})
}

func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		matched, err := filepath.Match(pattern, path)
		if err != nil {
			continue
		}

		if matched {
			return true
		}

		// Также проверяем, начинается ли путь с паттерна (для директорий)
		if strings.HasPrefix(path, pattern) {
			return true
		}
	}

	return false
}

func copyFile(src, dst string, mode os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

