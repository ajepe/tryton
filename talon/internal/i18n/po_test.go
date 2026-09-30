package i18n

import (
	"strings"
	"testing"
)

func TestParsePO(t *testing.T) {
	source := `
msgid ""
msgstr "Content-Type: text/plain; charset=utf-8\n"

msgctxt "field:party.party,name:"
msgid "Name"
msgstr "Nom"

#, fuzzy
msgctxt "field:party.party,code:"
msgid "Code"
msgstr "Code traduit"

msgctxt "field:party.party,address:"
msgid "Multi"
msgstr "Première ligne\nDeuxième ligne"
`
	entries, err := ParsePO(strings.NewReader(source))
	if err != nil {
		t.Fatalf("ParsePO() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("ParsePO() returned %d entries, want 4", len(entries))
	}
	if entries[1].Msgctxt != "field:party.party,name:" || entries[1].Msgstr != "Nom" {
		t.Errorf("entry = %+v", entries[1])
	}
	if !entries[2].Fuzzy {
		t.Error("entry 2 should be fuzzy")
	}
	if entries[3].Msgstr != "Première ligne\nDeuxième ligne" {
		t.Errorf("multiline = %q", entries[3].Msgstr)
	}
}

func TestParsePOContext(t *testing.T) {
	translation, err := parsePOContext(POEntry{
		Msgctxt: "field:party.party,name:testmod.party_1",
		Msgid:   "Name",
		Msgstr:  "Nom",
	})
	if err != nil {
		t.Fatalf("parsePOContext() error = %v", err)
	}
	if translation.Type != "field" || translation.Name != "party.party,name" {
		t.Errorf("translation = %+v", translation)
	}
	if translation.Resource != "testmod.party_1" {
		t.Errorf("resource = %q", translation.Resource)
	}

	if _, err := parsePOContext(POEntry{Msgctxt: "invalid"}); err == nil {
		t.Error("parsePOContext() expected error")
	}
}

func TestDecodePOString(t *testing.T) {
	if got := decodePOString(`"hello \n world"`); got != "hello \n world" {
		t.Errorf("decodePOString() = %q", got)
	}
}
