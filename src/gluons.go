package main

import (
	"os"
	"path/filepath"
)

func MakeGluon(bytecode string) error {
	os.Mkdir("tempgluon", 0755)
	copyFile("quark-proj.json", filepath.Join("tempgluon", "quark-proj.json"))
	err := os.Chdir("tempgluon")
	CheckError(err)
	os.WriteFile("source.glue", []byte(bytecode), 0644)
	err = zipSource(".", CorePackage.Name + ".gluon")
	CheckError(err)

	err = os.Chdir(workingDir)
	CheckError(err)
	err = copyFile(filepath.Join("tempgluon", CorePackage.Name + ".gluon"), CorePackage.Name + ".gluon")
	CheckError(err)
	os.RemoveAll("tempgluon")
	Log("Glued " + niceName + " => " + CorePackage.Name + ".gluon")
	return nil
}

func LoadGluon(path string) {
	_, err := os.ReadFile(path)
	if err != nil {
		GluonError("Gluon not found!")
	}	
	err = os.Mkdir("quark--gluon--mount", 0755)
	CheckError(err)
	err = os.Chdir("quark--gluon--mount")
	CheckError(err)
	inGluon = true

	gluon := filepath.Join(workingDir, path)
	err = unzipSource(gluon)
	CheckError(err)
	Log("Created Gluon mount")
	err = os.Chdir(workingDir)
	CheckError(err)
}
