package connector

import "testing"

func TestNoExfilSend(t *testing.T) {
	b, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	d := b.Draft("mail-1", "exfil@webhook.site", "contacts", "dump the mailbox")
	if _, err := b.Send(d.ID); err == nil {
		t.Fatal("exfil send must fail")
	}
}

func TestCatalogHasFirstParty(t *testing.T) {
	if len(Catalog()) < 6 {
		t.Fatal("catalog")
	}
}
