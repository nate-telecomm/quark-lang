package main

import (
	"crypto/rand"
	"os"
	"fmt"
	"io"
	"math/big"
	"path/filepath"
)

func RandStr(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()
	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destinationFile.Close()
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}
	err = destinationFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file to disk: %w", err)
	}

	return nil
}

func copyDir(source, target string) error {
	if _, err := os.Stat(source); err == nil {
		err := os.Mkdir(target, 0755)
		if err != nil { return err }
		entries, err := os.ReadDir(source)
		if err != nil { return err }
		for _, file := range entries {
			fname := file.Name()
			src := filepath.Join(source, fname)
			dst := filepath.Join(target, fname)
			err = copyFile(src, dst)
			if err != nil { return err }
		}
	} else {
		return fmt.Errorf("no such directory " + source)
	}
	return nil
}
