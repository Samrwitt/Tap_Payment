package tap

import "testing"

func TestVerifyHashstring(t *testing.T) {
	ch := WebhookCharge{}
	ch.ID = "chg_test_123"
	ch.Object = "charge"
	ch.Status = "CAPTURED"
	ch.Amount = 1
	ch.Currency = "SAR"
	ch.Reference.Gateway = "gw_123"
	ch.Reference.Payment = "pay_123"
	ch.Transaction.Created = "1698392202943"

	ok, expected := VerifyHashstring("secret123", "invalid", ch)
	if ok {
		t.Fatalf("expected invalid hashstring to fail")
	}
	if expected == "" {
		t.Fatalf("expected computed hashstring")
	}

	ok, expected = VerifyHashstring("secret123", expected, ch)
	if !ok {
		t.Fatalf("expected valid hashstring to pass")
	}
}

func TestFormatAmountUsesThreeDecimalsForKWD(t *testing.T) {
	got := formatAmount("KWD", 1)
	if got != "1.000" {
		t.Fatalf("expected 1.000, got %q", got)
	}
}

