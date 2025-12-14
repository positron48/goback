package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"backup-tool/compression"
	"backup-tool/config"
	"backup-tool/hooks"
	"backup-tool/retention"
	"backup-tool/utils"
)

type Executor struct {
	globalConfig *config.GlobalConfig
}

func NewExecutor(globalConfig *config.GlobalConfig) *Executor {
	return &Executor{
		globalConfig: globalConfig,
	}
}

func (e *Executor) ExecuteBackup(backupConfig *config.BackupConfig) error {
	utils.PrintHeader("Starting backup: %s", backupConfig.Name)

	// Выполняем локальные pre-hooks
	if len(backupConfig.PreHooks) > 0 {
		fmt.Printf("Running backup pre-hooks...\n")
		if err := hooks.RunHooks(backupConfig.PreHooks); err != nil {
			fmt.Printf("Warning: backup pre-hooks completed with errors\n")
		}
	}

	// Определяем тип сжатия
	compressionType := backupConfig.Compression
	if compressionType == "" {
		compressionType = e.globalConfig.DefaultCompression
	}

	// Создаем временную директорию для бэкапа
	tmpDir, err := os.MkdirTemp("", "backup-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var sourcePath string

	// Выполняем бэкап
	if backupConfig.SourceDir != "" {
		// Бэкап директории
		sourcePath = tmpDir
		if err := CopyDirectory(backupConfig.SourceDir, sourcePath, backupConfig.ExcludePatterns); err != nil {
			return fmt.Errorf("failed to copy directory: %w", err)
		}
	} else if backupConfig.Command != "" {
		// Бэкап через команду
		if err := ExecuteCommand(backupConfig.Command, backupConfig.OutputFile); err != nil {
			return fmt.Errorf("failed to execute command: %w", err)
		}

		// Копируем output_file во временную директорию
		sourcePath = filepath.Join(tmpDir, filepath.Base(backupConfig.OutputFile))
		if err := copyFileToTemp(backupConfig.OutputFile, sourcePath); err != nil {
			return fmt.Errorf("failed to copy output file: %w", err)
		}
	} else {
		return fmt.Errorf("invalid backup configuration: no source_dir or command")
	}

	// Создаем имя файла
	now := time.Now()
	filename := utils.GenerateFilename(e.globalConfig.FilenameMask, backupConfig.Name, now)
	ext := utils.GetExtension(compressionType)
	if ext != "" {
		filename += ext
	}

	// Создаем целевую директорию
	backupSubDir := filepath.Join(e.globalConfig.BackupDir, backupConfig.Subdirectory)
	if err := os.MkdirAll(backupSubDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	destinationPath := filepath.Join(backupSubDir, filename)

	// Применяем сжатие
	compressor, err := compression.NewCompressor(compressionType)
	if err != nil {
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	fmt.Printf("Compressing to %s...\n", destinationPath)
	if err := compressor.Compress(sourcePath, destinationPath); err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	utils.PrintSuccess("Backup created: %s", filename)

	// Применяем retention policy
	retentionPolicy := e.globalConfig.Retention
	if backupConfig.Retention != nil {
		retentionPolicy = *backupConfig.Retention
	}

	fmt.Printf("Applying retention policy...\n")
	if err := retention.ApplyRetention(e.globalConfig.BackupDir, backupConfig.Subdirectory, backupConfig.Name, retention.RetentionPolicy{
		Daily:   retentionPolicy.Daily,
		Weekly:  retentionPolicy.Weekly,
		Monthly: retentionPolicy.Monthly,
		Yearly:  retentionPolicy.Yearly,
	}); err != nil {
		fmt.Printf("Warning: retention policy failed: %v\n", err)
	}

	// Выполняем локальные post-hooks
	if len(backupConfig.PostHooks) > 0 {
		fmt.Printf("Running backup post-hooks...\n")
		if err := hooks.RunHooks(backupConfig.PostHooks); err != nil {
			fmt.Printf("Warning: backup post-hooks completed with errors\n")
		}
	}

	utils.PrintSuccess("Backup completed: %s", backupConfig.Name)
	return nil
}

func copyFileToTemp(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

