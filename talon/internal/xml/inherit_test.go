package xml

import (
	"strings"
	"testing"
)

func TestParseAndMarshalArch(t *testing.T) {
	source := `<form><field name="name"/><notebook><page string="More"/></notebook></form>`
	root, err := ParseArch(source)
	if err != nil {
		t.Fatalf("ParseArch() error = %v", err)
	}
	if root.Name != "form" || len(root.Children) != 2 {
		t.Fatalf("root = %+v", root)
	}
	output, err := MarshalArch(root)
	if err != nil {
		t.Fatalf("MarshalArch() error = %v", err)
	}
	if !strings.Contains(output, `name="name"`) {
		t.Errorf("output = %q", output)
	}
}

func TestFindXPath(t *testing.T) {
	root, _ := ParseArch(`<form><notebook><page name="general"/></notebook><field name="name"/></form>`)

	node, err := Find(root, "//field[@name='name']")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if node == nil || node.Attr("name") != "name" {
		t.Errorf("Find() = %+v", node)
	}

	node, err = Find(root, "//notebook/page[@name='general']")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if node == nil || node.Name != "page" {
		t.Errorf("Find() = %+v", node)
	}

	node, _ = Find(root, "//missing")
	if node != nil {
		t.Errorf("Find(missing) = %+v, want nil", node)
	}
}

func TestApplyInheritancePositions(t *testing.T) {
	base, _ := ParseArch(`<form><field name="a"/><field name="b"/></form>`)

	cases := []struct {
		name string
		arch string
		want []string
	}{
		{
			name: "after",
			arch: `<xpath expr="//field[@name='a']" position="after"><field name="c"/></xpath>`,
			want: []string{"a", "c", "b"},
		},
		{
			name: "before",
			arch: `<xpath expr="//field[@name='b']" position="before"><field name="c"/></xpath>`,
			want: []string{"a", "c", "b"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			root := base.Clone()
			extension, _ := ParseArch(tt.arch)
			if err := ApplyInheritance(root, extension); err != nil {
				t.Fatalf("ApplyInheritance() error = %v", err)
			}
			got := []string{}
			for _, child := range root.Children {
				got = append(got, child.Attr("name"))
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("children = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyInheritanceReplaceAndInside(t *testing.T) {
	root, _ := ParseArch(`<form><field name="a"/><notebook><page name="p"/></notebook></form>`)

	replace, _ := ParseArch(`<xpath expr="//field[@name='a']" position="replace"><field name="z"/></xpath>`)
	if err := ApplyInheritance(root, replace); err != nil {
		t.Fatalf("replace error = %v", err)
	}
	if root.Children[0].Attr("name") != "z" {
		t.Errorf("replace failed: %+v", root.Children[0])
	}

	inside, _ := ParseArch(`<xpath expr="//notebook" position="inside"><page name="extra"/></xpath>`)
	if err := ApplyInheritance(root, inside); err != nil {
		t.Fatalf("inside error = %v", err)
	}
	notebook := root.Children[1]
	if len(notebook.Children) != 2 || notebook.Children[1].Attr("name") != "extra" {
		t.Errorf("inside failed: %+v", notebook.Children)
	}
}

func TestApplyInheritanceAttributes(t *testing.T) {
	root, _ := ParseArch(`<form><field name="a"/></form>`)
	extension, _ := ParseArch(`<xpath expr="//field[@name='a']" position="attributes"><attribute name="readonly">1</attribute></xpath>`)
	if err := ApplyInheritance(root, extension); err != nil {
		t.Fatalf("attributes error = %v", err)
	}
	if root.Children[0].Attr("readonly") != "1" {
		t.Errorf("readonly = %q, want 1", root.Children[0].Attr("readonly"))
	}
}

func TestApplyInheritanceMissingTarget(t *testing.T) {
	root, _ := ParseArch(`<form><field name="a"/></form>`)
	extension, _ := ParseArch(`<xpath expr="//field[@name='missing']" position="after"><field name="b"/></xpath>`)
	if err := ApplyInheritance(root, extension); err == nil {
		t.Error("ApplyInheritance() expected error for missing target")
	}
}

func TestMergeViews(t *testing.T) {
	base := `<form><field name="a"/></form>`
	inherit := `<data><xpath expr="//field[@name='a']" position="after"><field name="b"/></xpath></data>`
	merged, err := MergeViews([]View{
		{ID: 1, Arch: base},
		{ID: 2, Arch: inherit, Inherit: 1, Priority: 10},
	})
	if err != nil {
		t.Fatalf("MergeViews() error = %v", err)
	}
	if !strings.Contains(merged, `name="b"`) {
		t.Errorf("merged = %q, want field b", merged)
	}
	indexA := strings.Index(merged, `name="a"`)
	indexB := strings.Index(merged, `name="b"`)
	if indexA > indexB {
		t.Errorf("field b should come after a: %q", merged)
	}
}
