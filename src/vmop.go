package main

import (
	"os"
	"strings"
	"path/filepath"
	"quark/vm"
	"slices"
)

func BuildGluon(projectDir string) error {
	var finished []byte
	processed := []string{}

	pkgsDir := filepath.Join(projectDir, "pkgs")
	if _, err := os.Stat(pkgsDir); err == nil {
		err := filepath.Walk(pkgsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".gluon") {
				LoadGluon(path)
				err := filepath.Walk("quark--gluon--mount", func(fpath string, finfo os.FileInfo, ferr error) error {
					if ferr != nil {
						return ferr
					}
					if !finfo.IsDir() && finfo.Name() == "source.glue" {
						codeBytes, err := os.ReadFile(fpath)
						if err != nil {
							return err
						}
						chk := Checksum(codeBytes)
						if slices.Contains(processed, chk) {
							return nil
						}
						finished = append(finished, codeBytes...)
						processed = append(processed, chk)
					}
					return nil
				})
				if err != nil {
					return err
				}
				os.RemoveAll("quark--gluon--mount")
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".quark") {
			codeBytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			chk := Checksum(codeBytes)
			if slices.Contains(processed, chk) {
				return nil
			}
			bc, err := vm.CompileSourceToBlob(string(codeBytes))
			if err != nil {
				return err
			}
			finished = append(finished, bc...)
			processed = append(processed, chk)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return MakeGluon(finished)
}
