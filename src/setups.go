package main

import (
	"encoding/json"
	"os"
	"github.com/nate-telecomm/go_ansi"
)

func SetupProj(name string) {
	CorePackage = &Package{
		Name: name,
	}
	data, _ := json.Marshal(CorePackage)
	os.WriteFile("quark-proj.json", data, 0644)
	os.Mkdir("pkgs", 0777)
	Log("Created new project " + ansi.Underline + CorePackage.Name + ansi.End)
}
