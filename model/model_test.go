package model

import "testing"

func TestNamedTypeString(t *testing.T) {
	cases := []struct {
		nt       NamedType
		expected string
	}{
		{
			NamedType{"", "Bar"},
			"Bar",
		},
		{
			NamedType{"foo", "Bar"},
			"Foo.Bar",
		},
	}

	pt := PackageTable{
		"foo": "Foo",
	}
	for _, tt := range cases {
		actual := tt.nt.String(pt)
		if actual != tt.expected {
			t.Errorf(`expected "%s" actual "%s"`, tt.expected, actual)
		}
	}
}

func TestPredeclaredTypeString(t *testing.T) {
	cases := []struct {
		pType    PredeclaredType
		expected string
	}{
		{
			PredeclaredType("string"),
			"string",
		},
		{
			PredeclaredType("int"),
			"int",
		},
		{
			PredeclaredType("bool"),
			"bool",
		},
		{
			PredeclaredType("float64"),
			"float64",
		},
		{
			PredeclaredType("error"),
			"error",
		},
		{
			PredeclaredType("Bar"),
			"Bar",
		},
	}

	pt := PackageTable{}
	for _, tt := range cases {
		actual := tt.pType.String(pt)
		if actual != tt.expected {
			t.Errorf(`expected "%s" actual "%s"`, tt.expected, actual)
		}
	}
}

func TestPredeclaredTypeAddPackagePaths(t *testing.T) {
	cases := []PredeclaredType{
		PredeclaredType("string"),
		PredeclaredType("int"),
		PredeclaredType("bool"),
		PredeclaredType("float64"),
		PredeclaredType("error"),
	}

	for _, pt := range cases {
		pps := PackagePathSet{}
		pt.addPackagePaths(pps)
		if len(pps) != 0 {
			t.Errorf("expected PackagePathSet is empty actual %#v", pps)
		}
	}
}
