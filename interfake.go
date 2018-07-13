package main

import (
	"flag"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	targetOption  = flag.String("target", "", "target interface")
	packageOption = flag.String("package", "", "package of the generated code")
	outputOption  = flag.String("output", "", "output file name")
)

func main() {
	var err error
	flag.Parse()

	if *targetOption == "" {
		log.Fatal("target option must be set")
	}

	files, err := parsePackageDir(".")
	intf, pkg := seekInterface(files, *targetOption)
	if intf == nil {
		log.Fatalf("not found interface %s", *targetOption)
	}

	outPackageName := *packageOption
	if outPackageName == "" {
		outPackageName = "fake_" + pkg
	}

	outPackagePath := packagePath(*outputOption)

	output := os.Stdout
	if *outputOption != "" {
		abs, err := filepath.Abs(*outputOption)
		if err != nil {
			log.Fatalf("failed identifying output parent directory: %v", err)
		}
		dir := filepath.Dir(abs)
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			log.Fatalf("failed making output parent directory: %v", err)
		}
		output, err = os.Create(abs)
		if err != nil {
			log.Fatalf("failed opening output file: %v", err)
		}
		defer output.Close()
	}

	g := NewGenerator()
	err = g.Generate(intf, outPackageName, outPackagePath)
	if err != nil {
		log.Fatalf("failed generating code: %v", err)
	}
	err = g.Format()
	if err != nil {
		log.Fatalf("failed formatting code: %v", err)
	}

	g.WriteTo(output)
}

func seekInterface(files []*GoFile, interfaceName string) (*Interface, string) {
	for _, f := range files {
		for _, i := range f.Interfaces {
			if i.Name == interfaceName {
				return i, f.PackageName
			}
		}
	}
	return nil, ""
}

func packagePath(outPath string) string {
	if outPath == "" {
		return ""
	}

	dst, _ := filepath.Abs(filepath.Dir(outPath))
	for _, prefix := range build.Default.SrcDirs() {
		if strings.HasPrefix(dst, prefix) {
			if rel, err := filepath.Rel(prefix, dst); err == nil {
				return rel
			}
		}
	}

	return ""
}
