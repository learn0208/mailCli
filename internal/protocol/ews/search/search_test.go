package search

import (
	"strings"
	"testing"

	"github.com/tschuyebuhl/ews"
)

func TestMarshalSubjectFindItem_matchesLibraryShape(t *testing.T) {
	body, err := marshalSubjectFindItem(Options{
		Subject: "更新状态摘要",
		Limit:   50,
		Folder:  "Inbox",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, `FieldURI="item:Subject"`) {
		t.Fatalf("missing subject field: %s", s)
	}
	if !strings.Contains(s, `Order="Descending"`) {
		t.Fatalf("expected SortOrder descending on DateTimeReceived: %s", s)
	}
	if strings.Contains(s, `message:From`) {
		t.Fatalf("must not request message:From in ItemShape: %s", s)
	}
	// Library struct order: ParentFolderIds before Restriction
	pi := strings.Index(s, "<m:ParentFolderIds>")
	ri := strings.Index(s, "<m:Restriction>")
	if pi < 0 || ri < 0 || pi > ri {
		t.Fatalf("expected ParentFolderIds before Restriction, got:\n%s", s)
	}
	_ = ews.BaseShapeIdOnly // compile-time link to ews types
}
