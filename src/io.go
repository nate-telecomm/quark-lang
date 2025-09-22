package main

import (
	"fmt"
	"github.com/nate-telecomm/go_ansi"
	"os"
)

func Cleanup() {
	Log("Cleaning up...")
	if inGluon {
		os.Chdir(workingDir)
		os.RemoveAll("quark--gluon--mount")
	}
}

func Exit(status int) {
	Cleanup()
	os.Exit(status)
}

func RuntimeError(msg string) {
	fmt.Println(ansi.Red + "------[RuntimeError]------" + ansi.End)
	fmt.Println(msg)
	fmt.Println(ansi.Red + "--------------------------" + ansi.End)
	Exit(1)
}
func RuntimeWarning(msg string) {
	fmt.Println(ansi.Yellow + "------[RuntimeWarning]------" + ansi.End)
	fmt.Println(msg)
	fmt.Println(ansi.Yellow + "--------------------------" + ansi.End)
}
func GluonError(msg string) {
	fmt.Println(ansi.Red + "------[GluonError]------" + ansi.End)
	fmt.Println(msg)
	fmt.Println(ansi.Red + "--------------------------" + ansi.End)
	Exit(1)
}
func GluonWarning(msg string) {
	fmt.Println(ansi.Yellow + "------[GluonWarning]------" + ansi.End)
	fmt.Println(msg)
	fmt.Println(ansi.Yellow + "--------------------------" + ansi.End)
}
func PackageError(msg string) {
	fmt.Println(ansi.Red + "------[PackageError]------" + ansi.End)
	fmt.Println(msg)
	fmt.Println(ansi.Red + "--------------------------" + ansi.End)
	Exit(1)
}
func PackageWarning(msg string) {
	fmt.Println(ansi.Yellow + "------[PackageWarning]------" + ansi.End)
	fmt.Println(msg)
	fmt.Println(ansi.Yellow + "--------------------------" + ansi.End)
}

func CheckError(e error) {
	if e != nil {
		fmt.Println(ansi.Red + "------[UnknownError]------" + ansi.End)
		Cleanup()
		fmt.Println("We will be panicing because it displays helpful debug info")
		fmt.Println("Hold on tight...")
		panic(e)
	}
}

func Log(msg string) {
	fmt.Println(ansi.Italic + "[LOG] " + ansi.End + msg)
}
