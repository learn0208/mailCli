package search

import (
	"encoding/xml"
	"testing"

	"github.com/tschuyebuhl/ews"
)

func TestParseGetItemMessages_multipleResponseMessages(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <m:GetItemResponse xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages" xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
      <m:ResponseMessages>
        <m:GetItemResponseMessage ResponseClass="Success">
          <m:Items>
            <t:Message>
              <t:ItemId Id="id-1" ChangeKey="ck-1"/>
              <t:Subject>one</t:Subject>
              <t:DateTimeReceived>2026-05-15T06:22:00Z</t:DateTimeReceived>
              <t:From><t:Mailbox><t:EmailAddress>a@example.com</t:EmailAddress></t:Mailbox></t:From>
              <t:HasAttachments>false</t:HasAttachments>
              <t:IsRead>false</t:IsRead>
            </t:Message>
          </m:Items>
        </m:GetItemResponseMessage>
        <m:GetItemResponseMessage ResponseClass="Success">
          <m:Items>
            <t:Message>
              <t:ItemId Id="id-2" ChangeKey="ck-2"/>
              <t:Subject>two</t:Subject>
              <t:DateTimeReceived>2026-05-15T07:22:00Z</t:DateTimeReceived>
              <t:Sender><t:Mailbox><t:EmailAddress>b@example.com</t:EmailAddress></t:Mailbox></t:Sender>
              <t:HasAttachments>true</t:HasAttachments>
              <t:IsRead>true</t:IsRead>
            </t:Message>
          </m:Items>
        </m:GetItemResponseMessage>
      </m:ResponseMessages>
    </m:GetItemResponse>
  </s:Body>
</s:Envelope>`)

	got, err := parseGetItemMessages(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].ItemID.Id != "id-1" || got[1].ItemID.Id != "id-2" {
		t.Fatalf("ids: %+v", got)
	}
	row := mergeDetail(Row{}, got[1])
	if row.From != "b@example.com" {
		t.Fatalf("sender from: %q", row.From)
	}
	_ = xml.Marshal
	_ = ews.BaseShapeDefault
}

func TestParseGetItemMessages_skipsErrorItem(t *testing.T) {
	raw := []byte(`<?xml version="1.0" encoding="utf-8"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <m:GetItemResponse xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages" xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
      <m:ResponseMessages>
        <m:GetItemResponseMessage ResponseClass="Error">
          <m:MessageText>stale</m:MessageText>
        </m:GetItemResponseMessage>
        <m:GetItemResponseMessage ResponseClass="Success">
          <m:Items>
            <t:Message>
              <t:ItemId Id="id-2"/>
              <t:DateTimeReceived>2026-05-15T07:22:00Z</t:DateTimeReceived>
            </t:Message>
          </m:Items>
        </m:GetItemResponseMessage>
      </m:ResponseMessages>
    </m:GetItemResponse>
  </s:Body>
</s:Envelope>`)
	got, err := parseGetItemMessages(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ItemID.Id != "id-2" {
		t.Fatalf("got %+v", got)
	}
}

func TestRowNeedsEnrich(t *testing.T) {
	if !rowNeedsEnrich(Row{}) {
		t.Fatal("empty row needs enrich")
	}
	if !rowNeedsEnrich(Row{From: "a@b.com"}) {
		t.Fatal("from without recipients still needs GetItem for To/time")
	}
	if !rowNeedsEnrich(Row{From: "a@b.com", DateTimeReceived: "2026-05-15T00:00:00Z"}) {
		t.Fatal("from+time but no recipients still needs enrich")
	}
	if rowNeedsEnrich(Row{From: "a@b.com", DateTimeReceived: "2026-05-15T00:00:00Z", To: []string{"x@y.z"}}) {
		t.Fatal("complete row should not need enrich")
	}
}

func TestMergeDetail_senderFallback(t *testing.T) {
	row := mergeDetail(Row{}, messageDetail{
		Sender: addressBlock{Mailbox: []struct {
			Name         string `xml:"Name"`
			EmailAddress string `xml:"EmailAddress"`
		}{{EmailAddress: "dev@zhaopin.com"}}},
		DateTimeReceived: "2026-05-15T06:22:00Z",
		HasAttachments:   "false",
		IsRead:           "true",
	})
	if row.From != "dev@zhaopin.com" {
		t.Fatalf("from=%q", row.From)
	}
	if !row.isReadKnown || row.IsRead != true {
		t.Fatalf("read=%v known=%v", row.IsRead, row.isReadKnown)
	}
}
