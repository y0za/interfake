// Copyright 2012 Google Inc.
// https://github.com/golang/mock/blob/master/mockgen/model/parse.go
// This file contains copies and modifications.
// Originaly under the Apache License, Version 2.0.
package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/y0za/interfake/model"
)

type fileParser struct {
	fileSet *token.FileSet
	imports map[string]string // package name => import path
}

type namedInterface struct {
	name *ast.Ident
	it   *ast.InterfaceType
}

func parsePackageDir(dir string) ([]*model.GoFile, error) {
	pkg, err := build.Default.ImportDir(dir, 0)
	if err != nil {
		return nil, err
	}

	var names []string
	names = append(names, pkg.GoFiles...)
	names = append(names, pkg.CgoFiles...)
	names = prefixFilesDir(dir, names)

	return parseFiles(names, pkg.ImportPath)
}

// prefixFilesDir places the directory name on the beginning of each file name in the list.
func prefixFilesDir(dir string, names []string) []string {
	if dir == "." {
		return names
	}
	ret := make([]string, len(names))
	for i, name := range names {
		ret[i] = filepath.Join(dir, name)
	}
	return ret
}

func parseFiles(names []string, pkg string) ([]*model.GoFile, error) {
	var goFiles []*model.GoFile

	for _, name := range names {
		if !strings.HasSuffix(name, ".go") {
			continue
		}

		fs := token.NewFileSet()
		p := fileParser{
			fileSet: fs,
			imports: make(map[string]string),
		}

		file, err := parser.ParseFile(fs, name, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("failed parsing source file %v: %v", name, err)
		}

		gf, err := p.parseFile(file, pkg)
		if err != nil {
			return nil, err
		}

		goFiles = append(goFiles, gf)
	}

	return goFiles, nil
}

func (p *fileParser) parseFile(file *ast.File, pkg string) (*model.GoFile, error) {
	var err error

	p.imports, err = importsOfFile(file)
	if err != nil {
		return nil, err
	}

	var is []*model.Interface
	for _, ni := range interfacesOfFile(file) {
		i, err := p.parseInterface(ni.name.String(), pkg, ni.it)
		if err != nil {
			return nil, err
		}
		is = append(is, i)
	}

	return &model.GoFile{
		PackageName: file.Name.String(),
		Interfaces:  is,
	}, nil
}

func (p *fileParser) parseInterface(name, pkg string, it *ast.InterfaceType) (*model.Interface, error) {
	intf := &model.Interface{Name: name}
	for _, field := range it.Methods.List {
		switch v := field.Type.(type) {
		case *ast.FuncType:
			if nn := len(field.Names); nn != 1 {
				return nil, fmt.Errorf("expected one name for interface %v, got %d", intf.Name, nn)
			}
			m := &model.Method{
				Name: field.Names[0].String(),
			}
			var err error
			m.Args, m.Results, err = p.parseFunc(pkg, v)
			if err != nil {
				return nil, err
			}
			intf.Methods = append(intf.Methods, m)
		default:
			return nil, fmt.Errorf("don't know how to mock method of type %T", field.Type)
		}
	}
	return intf, nil
}

func (p *fileParser) parseFunc(pkg string, f *ast.FuncType) (args []*model.Parameter, results []*model.Parameter, err error) {
	if f.Params != nil {
		args, err = p.parseFieldList(pkg, f.Params.List)
		if err != nil {
			return nil, nil, p.errorf(f.Pos(), "failed parsing arguments: %v", err)
		}
	}
	if f.Results != nil {
		results, err = p.parseFieldList(pkg, f.Results.List)
		if err != nil {
			return nil, nil, p.errorf(f.Pos(), "failed parsing results: %v", err)
		}
	}
	return
}

func (p *fileParser) parseFieldList(pkg string, fields []*ast.Field) ([]*model.Parameter, error) {
	var ps []*model.Parameter
	for _, f := range fields {
		t, err := p.parseType(pkg, f.Type)
		if err != nil {
			return nil, err
		}

		if len(f.Names) == 0 {
			// anonymous arg
			ps = append(ps, &model.Parameter{Type: t})
			continue
		}
		for _, name := range f.Names {
			ps = append(ps, &model.Parameter{Name: name.Name, Type: t})
		}
	}
	return ps, nil
}

func (p *fileParser) parseType(pkg string, typ ast.Expr) (model.Type, error) {
	switch v := typ.(type) {
	case *ast.ArrayType:
		ln := -1
		if v.Len != nil {
			x, err := strconv.Atoi(v.Len.(*ast.BasicLit).Value)
			if err != nil {
				return nil, p.errorf(v.Len.Pos(), "bad array size: %v", err)
			}
			ln = x
		}
		t, err := p.parseType(pkg, v.Elt)
		if err != nil {
			return nil, err
		}
		if ln == -1 {
			return &model.SliceType{Type: t}, nil
		} else {
			return &model.ArrayType{Len: ln, Type: t}, nil
		}
	case *ast.ChanType:
		t, err := p.parseType(pkg, v.Value)
		if err != nil {
			return nil, err
		}
		var dir model.ChanDir
		if v.Dir == ast.SEND {
			dir = model.SendDirection
		}
		if v.Dir == ast.RECV {
			dir = model.RecvDirection
		}
		return &model.ChanType{Direction: dir, Type: t}, nil
	case *ast.FuncType:
		args, results, err := p.parseFunc(pkg, v)
		if err != nil {
			return nil, err
		}
		return &model.FuncType{Args: args, Results: results}, nil
	case *ast.Ident:
		if v.IsExported() {
			// `pkg` may be an aliased imported pkg
			// if so, patch the import w/ the fully qualified import
			maybeImportedPkg, ok := p.imports[pkg]
			if ok {
				pkg = maybeImportedPkg
			}
			// assume type in this package
			return &model.NamedType{Package: pkg, Type: v.Name}, nil
		} else {
			// assume predeclared type
			return model.PredeclaredType(v.Name), nil
		}
	case *ast.InterfaceType:
		if v.Methods != nil && len(v.Methods.List) > 0 {
			return nil, p.errorf(v.Pos(), "can't handle non-empty unnamed interface types")
		}
		return model.PredeclaredType("interface{}"), nil
	case *ast.MapType:
		key, err := p.parseType(pkg, v.Key)
		if err != nil {
			return nil, err
		}
		value, err := p.parseType(pkg, v.Value)
		if err != nil {
			return nil, err
		}
		return &model.MapType{Key: key, Value: value}, nil
	case *ast.SelectorExpr:
		pkgName := v.X.(*ast.Ident).String()
		pkg, ok := p.imports[pkgName]
		if !ok {
			return nil, p.errorf(v.Pos(), "unknown package %q", pkgName)
		}
		return &model.NamedType{Package: pkg, Type: v.Sel.String()}, nil
	case *ast.StarExpr:
		t, err := p.parseType(pkg, v.X)
		if err != nil {
			return nil, err
		}
		return &model.PointerType{Type: t}, nil
	case *ast.StructType:
		if v.Fields != nil && len(v.Fields.List) > 0 {
			return nil, p.errorf(v.Pos(), "can't handle non-empty unnamed struct types")
		}
		return model.PredeclaredType("struct{}"), nil
	}

	return nil, fmt.Errorf("don't know how to parse type %T", typ)
}

func (p *fileParser) errorf(pos token.Pos, format string, args ...interface{}) error {
	ps := p.fileSet.Position(pos)
	format = "%s:%d:%d: " + format
	args = append([]interface{}{ps.Filename, ps.Line, ps.Column}, args...)
	return fmt.Errorf(format, args...)
}

// importsOfFile returns a map of package name to import path
// of the imports in file.
func importsOfFile(file *ast.File) (map[string]string, error) {
	m := make(map[string]string)
	for _, is := range file.Imports {
		var pkgName string
		importPath := is.Path.Value[1 : len(is.Path.Value)-1] // remove quotes

		if is.Name != nil {
			// Named imports are always certain.
			if is.Name.Name == "_" {
				continue
			}
			pkgName = is.Name.Name
		} else {
			pkg, err := build.Import(importPath, "", 0)
			if err != nil {
				// Fallback to import path suffix. Note that this is uncertain.
				_, last := path.Split(importPath)
				// If the last path component has dots, the first dot-delimited
				// field is used as the name.
				pkgName = strings.SplitN(last, ".", 2)[0]
			} else {
				pkgName = pkg.Name
			}
		}

		if _, ok := m[pkgName]; ok {
			return m, fmt.Errorf("imported package collision: %q imported twice", pkgName)
		}
		m[pkgName] = importPath
	}
	return m, nil
}

func interfacesOfFile(file *ast.File) []namedInterface {
	var nis []namedInterface

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			it, ok := ts.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			nis = append(nis, namedInterface{ts.Name, it})
		}
	}

	return nis
}
