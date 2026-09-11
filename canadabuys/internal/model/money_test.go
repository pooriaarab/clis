package model

import "testing"

func TestParseCents(t *testing.T) {
	cases := []struct {
		in    string
		cents int64
		kind  string
	}{
		{"", 0, AmountBlank}, {"   ", 0, AmountBlank},
		{"0", 0, AmountZero}, {"0.00", 0, AmountZero}, {"-0.00", 0, AmountZero},
		{"-150.00", -15000, AmountNegative}, {"-0.50", -50, AmountNegative},
		{"215000.00", 21500000, AmountPositive}, {"1500000", 150000000, AmountPositive},
		{"1.00", 100, AmountPositive}, {"12.5", 1250, AmountPositive},
		{"13044299191.04", 1304429919104, AmountPositive},
		{"abc", 0, AmountInvalid}, {"1,000.00", 0, AmountInvalid},
		{"$5.00", 0, AmountInvalid}, {"1.234", 0, AmountInvalid},
		{"99999999999999999999999.00", 0, AmountInvalid},
		{"92233720368547759", 0, AmountInvalid},
	}
	for _, c := range cases {
		if got, kind := ParseCents(c.in); got != c.cents || kind != c.kind {
			t.Errorf("ParseCents(%q) = (%d, %q), want (%d, %q)", c.in, got, kind, c.cents, c.kind)
		}
	}
}

func TestCategoryShare(t *testing.T) {
	if got := CategoryShare(500, 1000); got != 0.5 {
		t.Fatalf("share = %v, want 0.5", got)
	}
	if got := CategoryShare(100, 0); got != 0 {
		t.Fatalf("zero category total gave %v, want 0", got)
	}
}
