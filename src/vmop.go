package main

import (
	"os"
	"strings"
	"path/filepath"
	"quark/vm"
	"slices"
	"encoding/json"
	"github.com/nate-telecomm/go_ansi"
)

func BuildGluon(projectDir string) error {
	finished := ""
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
							if slices.Contains(processed, Checksum(codeBytes)) { return nil }
							finished += string(codeBytes) + "\n"
							processed = append(processed, Checksum(codeBytes))
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
			if slices.Contains(processed, Checksum(codeBytes)) { return nil }
			bc := vm.ToBytecode(string(codeBytes))
			finished += bc + "\n"
			processed = append(processed, Checksum(codeBytes))
		}
		return nil
	})
	if err != nil {
		return err
	}
	return MakeGluon(finished)
}

func Run(gluon string) {
	err := os.Mkdir("temprunning", 0755)
	CheckError(err)
	err = copyFile(gluon, filepath.Join("temprunning", gluon))
	CheckError(err)
	err = os.Chdir("temprunning")
	CheckError(err)
	unzipSource(gluon)
	os.Remove(gluon)

	var p Package
	data, _ := os.ReadFile("quark-proj.json")
	json.Unmarshal(data, &p)
	CorePackage = &p
	niceName = ansi.Underline + CorePackage.Name + ansi.End
	Log("Running " + niceName)
}
