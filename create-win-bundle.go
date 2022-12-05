package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	u "github.com/sunshine69/golang-tools/utils"
	cp "github.com/otiai10/copy"
)

// Python and msys shell is like s***t. File not found while file exists and etc etc.. FFS lets write it in golang
func CreateWinBundle(mingw64Prefix string) {
	srcDir, err := os.Getwd()
	u.CheckErr(err, "Getwd")
	const BINARY_NAME = "fltkeditor"
	srcRootDir := filepath.Dir(srcDir)
	targetDir := srcRootDir + "/" + BINARY_NAME + "-windows-bundle"

	os.RemoveAll(targetDir)
	for _, _f := range []string{"/bin", "/lib", "/share"} {
		os.MkdirAll(targetDir+_f, 0755)
	}

	// err = cp.Copy(mingw64Prefix+"/lib/gdk-pixbuf-2.0", targetDir+"/lib/gdk-pixbuf-2.0")
	// fmt.Println(err)

	exeFiles, err := filepath.Glob(srcDir + "/" + BINARY_NAME + "*.exe")
	u.CheckErr(err, "Glob")
	for _, _f := range exeFiles {
		cp.Copy(_f, targetDir+"/bin/"+filepath.Base(_f))
	}

	dllFilesByte, err := os.ReadFile(srcDir + "/dll_files.lst")
	u.CheckErr(err, "dll_files")
	dllFilesStr := string(dllFilesByte)
	dllFilesStr = strings.ReplaceAll(dllFilesStr, "\r\n", "\n")
	lines := strings.Split(dllFilesStr, "\n")
	for _, _f := range lines {
		if _f != "" {
			fmt.Printf("Copy %s/bin/%s => %s/%s\n", mingw64Prefix, _f, targetDir+"/bin", _f)
			err = cp.Copy(mingw64Prefix+"/bin/"+_f, targetDir+"/bin/"+_f)
			fmt.Println(err)
		}
	}
	fmt.Println("Output folder: " + targetDir)
}
