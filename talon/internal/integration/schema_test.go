package integration

import (
	"testing"

	"github.com/ajepe/talon/internal/testsupport"
)

func TestSchemaCreatesTables(t *testing.T) {
	db := testsupport.OpenDB(t)

	for _, table := range []string{
		"res_user",
		"ir_sequence",
		"party_party",
		"party_address",
		"product_template",
		"account_move",
		"stock_move",
		"sale_sale",
		"purchase_purchase",
	} {
		if !testsupport.TableExists(t, db, table) {
			t.Errorf("table %s missing", table)
		}
	}
	if !testsupport.TableExists(t, db, "party_party-party_category") {
		t.Error("m2m table party_party-party_category missing")
	}
	if !testsupport.ColumnExists(t, db, "party_party", "create_uid") {
		t.Error("column party_party.create_uid missing")
	}
	if !testsupport.ColumnExists(t, db, "party_party", "_timestamp") {
		t.Error("column party_party._timestamp missing")
	}
}
