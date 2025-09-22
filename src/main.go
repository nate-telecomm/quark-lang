package main

import (
	"strings"
	"os"
	"github.com/nate-telecomm/go_ansi"
	"encoding/json"
)

func Init() {
	// check for project file
	data, err := os.ReadFile("quark-proj.json")
	if err != nil {
		PackageError("No Quark package file found!")
	}
	if _, err = os.Stat("pkgs"); err != nil {
		PackageError("No pkgs directory found!")
	}
	var p Package
	err = json.Unmarshal(data, &p)
	if err != nil {
		PackageError("JSON decoding error: " + err.Error())
	}
	CorePackage = &p
	niceName = ansi.Underline + CorePackage.Name + ansi.End
	Log("Using package " + niceName)
}

func main() {
	var weREALLYneedtocleanup bool = false
	// step 1. setup vars
	osArgs = os.Args[1:]
	if _, err := os.Stat("/data/data/com.termux"); err == nil {
		// termux, which go sucks ass at doing arguments for
		osArgs = osArgs[1:]
	}

	dir, err := os.Getwd()
	CheckError(err)
	workingDir = dir

	// step 2. handle first args
	if len(osArgs) < 1 { RuntimeError("No arguments provided!") }
	if osArgs[0] == "new" {
		if len(osArgs) != 2 {
			PackageError("No package name provided!")
		}
		SetupProj(osArgs[1])
	} else if osArgs[0] == "superglue" {
		if len(osArgs) != 2 {
			GluonError("No Gluon provided!")
		}
		if !strings.HasSuffix(osArgs[1], ".gluon") {
			GluonWarning("This file does not have the .gluon extension, it might not be a Gluon")
		}
		Run(osArgs[1])
	} else {
		Init()
		switch osArgs[0] {
		case "glue":
			if len(osArgs) != 2 {
				RuntimeError("glue takes one argument!")
			}
			if osArgs[1] == "this" {
				err = BuildGluon(".")
				if err != nil {
					GluonError(err.Error())
				}
			}
		}
	}
	// step 3. cleanup
	if weREALLYneedtocleanup {
		Cleanup()
	}
}
