package compression

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Compressor interface {
	Compress(source, destination string) error
}

type GzipCompressor struct{}

func (c *GzipCompressor) Compress(source, destination string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	writer := gzip.NewWriter(dstFile)
	defer writer.Close()

	_, err = io.Copy(writer, srcFile)
	if err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	return nil
}

type ZipCompressor struct{}

func (c *ZipCompressor) Compress(source, destination string) error {
	zipFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	// Если source - это файл
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if !info.IsDir() {
		return c.addFileToZip(writer, source, filepath.Base(source))
	}

	// Если source - это директория
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории
		if info.IsDir() {
			return nil
		}

		// Проверяем, является ли это симлинком, указывающим на директорию
		if info.Mode()&os.ModeSymlink != 0 {
			// Проверяем, куда указывает симлинк
			target, err := os.Readlink(path)
			if err != nil {
				// Не удалось прочитать симлинк, пропускаем
				return nil
			}
			// Получаем абсолютный путь цели
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}
			// Проверяем, является ли цель директорией
			if targetInfo, err := os.Stat(target); err == nil && targetInfo.IsDir() {
				// Симлинк указывает на директорию, пропускаем
				return nil
			}
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		return c.addFileToZip(writer, path, relPath)
	})
}

func (c *ZipCompressor) addFileToZip(writer *zip.Writer, filePath, zipPath string) error {
	// Проверяем, что это файл, а не директория
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Дополнительная проверка: если это директория, пропускаем
	if info.IsDir() {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = zipPath
	header.Method = zip.Deflate

	w, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, file)
	return err
}

type TarCompressor struct{}

func (c *TarCompressor) Compress(source, destination string) error {
	tarFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer tarFile.Close()

	writer := tar.NewWriter(tarFile)
	defer writer.Close()

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if !info.IsDir() {
		return c.addFileToTar(writer, source, filepath.Base(source))
	}

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		return c.addFileToTar(writer, path, relPath)
	})
}

func (c *TarCompressor) addFileToTar(writer *tar.Writer, filePath, tarPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	header.Name = tarPath

	if err := writer.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

type TarGzCompressor struct{}

func (c *TarGzCompressor) Compress(source, destination string) error {
	// Сначала создаем tar во временный файл
	tmpTar := destination + ".tmp.tar"
	if err := (&TarCompressor{}).Compress(source, tmpTar); err != nil {
		return err
	}
	defer os.Remove(tmpTar)

	// Затем сжимаем gzip
	tarFile, err := os.Open(tmpTar)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer tarFile.Close()

	gzFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create gzip file: %w", err)
	}
	defer gzFile.Close()

	writer := gzip.NewWriter(gzFile)
	defer writer.Close()

	_, err = io.Copy(writer, tarFile)
	if err != nil {
		return fmt.Errorf("failed to compress tar: %w", err)
	}

	return nil
}

type NoCompressor struct{}

func (c *NoCompressor) Compress(source, destination string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func NewCompressor(compressionType string) (Compressor, error) {
	switch strings.ToLower(compressionType) {
	case "gzip":
		return &GzipCompressor{}, nil
	case "zip":
		return &ZipCompressor{}, nil
	case "tar":
		return &TarCompressor{}, nil
	case "tar.gz":
		return &TarGzCompressor{}, nil
	case "none", "":
		return &NoCompressor{}, nil
	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType)
	}
}

