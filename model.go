// Copyright 2012 Google Inc.
// https://github.com/golang/mock/blob/master/mockgen/model/model.go
// This file contains copies and modifications.
// Originaly under the Apache License, Version 2.0.
package main

import (
	"fmt"
	"io"
	"strings"
)

// PackageTable represents correspondance of package path and package name.
// key: package path
// value: package name
type PackageTable map[string]string

// PackagePathSet records package path which is used.
// key: package path
type PackagePathSet map[string]struct{}

// GoFile is a .go file.
type GoFile struct {
	PackageName string
	Interfaces  []*Interface
}

func (gf *GoFile) Print(w io.Writer) {
	fmt.Fprintf(w, "go file package %s\n", gf.PackageName)
	for _, intf := range gf.Interfaces {
		intf.Print(w)
	}
}

// Interface is a Go interface.
type Interface struct {
	Name    string
	Methods []*Method
}

func (intf *Interface) Print(w io.Writer) {
	fmt.Fprintf(w, "interface %s\n", intf.Name)
	for _, m := range intf.Methods {
		m.Print(w)
	}
}

func (intf *Interface) PackagePaths() PackagePathSet {
	pps := make(PackagePathSet)
	for _, method := range intf.Methods {
		method.addPackagePaths(pps)
	}
	return pps
}

// Method is a single method of an interface.
type Method struct {
	Name    string
	Args    []*Parameter
	Results []*Parameter
}

func (m *Method) Print(w io.Writer) {
	fmt.Fprintf(w, "  - method %s\n", m.Name)
	if len(m.Args) > 0 {
		fmt.Fprintf(w, "    args:\n")
		for _, p := range m.Args {
			p.Print(w)
		}
	}
	if len(m.Results) > 0 {
		fmt.Fprintf(w, "    results:\n")
		for _, p := range m.Results {
			p.Print(w)
		}
	}
}

func (m *Method) addPackagePaths(pps PackagePathSet) {
	for _, p := range m.Args {
		p.Type.addPackagePaths(pps)
	}
	for _, p := range m.Results {
		p.Type.addPackagePaths(pps)
	}
}

// Parameter is an argument or return parameter of a method.
type Parameter struct {
	Name string // may be empty
	Type Type
}

func (p *Parameter) Print(w io.Writer) {
	n := p.Name
	if n == "" {
		n = `""`
	}
	fmt.Fprintf(w, "    - %v: %v\n", n, p.Type.String(nil))
}

type Type interface {
	String(pt PackageTable) string
	addPackagePaths(pps PackagePathSet)
}

type ArrayType struct {
	Len  int
	Type Type
}

func (at *ArrayType) String(pt PackageTable) string {
	return fmt.Sprintf("[%d]", at.Len) + at.Type.String(pt)
}

func (at *ArrayType) addPackagePaths(pps PackagePathSet) {
	at.Type.addPackagePaths(pps)
}

type SliceType struct {
	Type Type
}

func (st *SliceType) String(pt PackageTable) string {
	return "[]" + st.Type.String(pt)
}

func (st *SliceType) addPackagePaths(pps PackagePathSet) {
	st.Type.addPackagePaths(pps)
}

// ChanType is a channel type.
type ChanType struct {
	Direction ChanDir
	Type      Type
}

// ChanDiris a channel direction.
type ChanDir int

const (
	SendDirection ChanDir = 1 << iota
	RecvDirection
)

func (ct *ChanType) String(pt PackageTable) string {
	s := ct.Type.String(pt)
	if ct.Direction == RecvDirection {
		return "<-chan " + s
	}
	if ct.Direction == SendDirection {
		return "chan<- " + s
	}
	return "chan " + s
}

func (ct *ChanType) addPackagePaths(pps PackagePathSet) {
	ct.Type.addPackagePaths(pps)
}

// FuncType is a function type.
type FuncType struct {
	Args    []*Parameter
	Results []*Parameter
}

func (ft *FuncType) String(pt PackageTable) string {
	args := make([]string, len(ft.Args))
	for i, p := range ft.Args {
		args[i] = p.Type.String(pt)
	}

	results := make([]string, len(ft.Results))
	for i, p := range ft.Results {
		results[i] = p.Type.String(pt)
	}
	resultsStr := strings.Join(results, ", ")
	if rc := len(ft.Results); rc == 1 {
		resultsStr = " " + resultsStr
	} else if rc > 1 {
		resultsStr = " (" + resultsStr + ")"
	}

	return "func(" + strings.Join(args, ", ") + ")" + resultsStr
}

func (ft *FuncType) addPackagePaths(pps PackagePathSet) {
	for _, p := range ft.Args {
		p.Type.addPackagePaths(pps)
	}
	for _, p := range ft.Results {
		p.Type.addPackagePaths(pps)
	}
}

// MapType is a map type.
type MapType struct {
	Key   Type
	Value Type
}

func (mt *MapType) String(pt PackageTable) string {
	return "map[" + mt.Key.String(pt) + "]" + mt.Value.String(pt)
}

func (mt *MapType) addPackagePaths(pps PackagePathSet) {
	mt.Key.addPackagePaths(pps)
	mt.Value.addPackagePaths(pps)
}

// NamedType is an exported type in a package.
type NamedType struct {
	Package string // may be empty
	Type    string
}

func (nt *NamedType) String(pt PackageTable) string {
	if nt.Package == "" {
		return nt.Type
	}
	return pt[nt.Package] + "." + nt.Type
}

func (nt *NamedType) addPackagePaths(pps PackagePathSet) {
	if nt.Package != "" {
		pps[nt.Package] = struct{}{}
	}
}

// PointerType is a pointer to another type.
type PointerType struct {
	Type Type
}

func (pType *PointerType) String(pt PackageTable) string {
	return "*" + pType.Type.String(pt)
}

func (pt *PointerType) addPackagePaths(pps PackagePathSet) {
	pt.Type.addPackagePaths(pps)
}

// PredeclaredType is a predeclared type such as "int".
type PredeclaredType string

func (pType PredeclaredType) String(pt PackageTable) string {
	return string(pType)
}

func (_ PredeclaredType) addPackagePaths(pps PackagePathSet) {
}
