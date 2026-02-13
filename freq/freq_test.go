package freq_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SladkyCitron/resona/freq"
)

func TestFrequency_String(t *testing.T) {
	tests := []struct {
		str string
		f   freq.Frequency
	}{
		{"0nHz", 0},
		{"1nHz", 1 * freq.NanoHertz},
		{"100.000µHz", 100 * freq.MicroHertz},
		{"1.100000µHz", 1100 * freq.NanoHertz},
		{"100.000mHz", 100 * freq.MilliHertz},
		{"2.200000mHz", 2200 * freq.MicroHertz},
		{"123.456789000mHz", 123456789},
		{"3.300Hz", 3300 * freq.MilliHertz},
		{"42Hz", 42 * freq.Hertz},
		{"4.234567Hz", 4234567 * freq.MicroHertz},
		{"123.456789123Hz", 123456789123},
		{"4.005kHz", 4*freq.KiloHertz + 5*freq.Hertz},
		{"4.005001kHz", 4*freq.KiloHertz + 5001*freq.MilliHertz},
		{"5.006MHz", 5*freq.MegaHertz + 6*freq.KiloHertz},
		{"8.000001MHz", 8*freq.MegaHertz + 1*freq.Hertz},
		{"2.400GHz", 2400 * freq.MegaHertz},
		{"8kHz", 8 * freq.KiloHertz},
		{"22.500kHz", 22500 * freq.Hertz},
		{"32kHz", 32 * freq.KiloHertz},
		{"44.100kHz", 44100 * freq.Hertz},
		{"48kHz", 48 * freq.KiloHertz},
		{"Inf Hz", 1<<63 - 1},
		{"-Inf Hz", -1 << 63},
	}

	for _, test := range tests {
		if str := test.f.String(); str != test.str {
			t.Errorf("Frequency(%d).String() = %s, want %s", int64(test.f), str, test.str)
		}
		if test.f > 0 {
			if str := (-test.f).String(); str != "-"+test.str {
				t.Errorf("Frequency(%d).String() = %s, want %s", int64(-test.f), str, "-"+test.str)
			}
		}
	}
}

func TestFrequency_NanoHertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect int64
	}{
		{-1000, -1000},
		{-1, -1},
		{1, 1},
		{1000, 1000},
	}

	for _, test := range tests {
		if actual := test.f.NanoHertz(); actual != test.expect {
			t.Errorf("Frequency(%v).NanoHertz() = %d, want %d", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_MicroHertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect int64
	}{
		{-1000, -1},
		{1000, 1},
	}

	for _, test := range tests {
		if actual := test.f.MicroHertz(); actual != test.expect {
			t.Errorf("Frequency(%v).MicroHertz() = %d, want %d", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_MilliHertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect int64
	}{
		{-1000000, -1},
		{1000000, 1},
	}

	for _, test := range tests {
		if actual := test.f.MilliHertz(); actual != test.expect {
			t.Errorf("Frequency(%v).MilliHertz() = %d, want %d", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_Hertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect float64
	}{
		{-1000000000, -1},
		{-100000000, -0.1},
		{-1, -1 / 1e9},
		{1, 1 / 1e9},
		{100000000, 0.1},
		{1000000000, 1},
	}

	for _, test := range tests {
		if actual := test.f.Hertz(); actual != test.expect {
			t.Errorf("Frequency(%v).Hertz() = %f, want %f", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_KiloHertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect float64
	}{
		{-1000000000000, -1},
		{-100000000000, -0.1},
		{-1, -1 / 1e12},
		{1, 1 / 1e12},
		{100000000000, 0.1},
		{1000000000000, 1},
	}

	for _, test := range tests {
		if actual := test.f.KiloHertz(); actual != test.expect {
			t.Errorf("Frequency(%v).KiloHertz() = %f, want %f", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_MegaHertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect float64
	}{
		{-1000000000000000, -1},
		{-100000000000000, -0.1},
		{-1, -1 / 1e15},
		{1, 1 / 1e15},
		{100000000000000, 0.1},
		{1000000000000000, 1},
	}

	for _, test := range tests {
		if actual := test.f.MegaHertz(); actual != test.expect {
			t.Errorf("Frequency(%v).MegaHertz() = %f, want %f", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_GigaHertz(t *testing.T) {
	tests := []struct {
		f      freq.Frequency
		expect float64
	}{
		{-1000000000000000000, -1},
		{-100000000000000000, -0.1},
		{-1, -1 / 1e18},
		{1, 1 / 1e18},
		{100000000000000000, 0.1},
		{1000000000000000000, 1},
	}

	for _, test := range tests {
		if actual := test.f.GigaHertz(); actual != test.expect {
			t.Errorf("Frequency(%v).GigaHertz() = %f, want %f", test.f, actual, test.expect)
		}
	}
}

func TestFrequency_Truncate(t *testing.T) {
	tests := []struct {
		f, m, expect freq.Frequency
	}{
		{0, freq.Hertz, 0},
		{freq.KiloHertz, -7 * freq.Hertz, freq.KiloHertz},
		{freq.KiloHertz, 0, freq.KiloHertz},
		{freq.KiloHertz, 1, freq.KiloHertz},
	}

	for _, test := range tests {
		if actual := test.f.Truncate(test.m); actual != test.expect {
			t.Errorf("Frequency(%v).Truncate(%v) = %v, want %v", test.f, test.m, actual, test.expect)
		}
	}
}

func TestFrequency_Round(t *testing.T) {
	tests := []struct {
		f, m, expect freq.Frequency
	}{
		{0, freq.Hertz, 0},
		{freq.KiloHertz, -7 * freq.Hertz, freq.KiloHertz},
		{-freq.KiloHertz, -7 * freq.Hertz, -freq.KiloHertz},
		{freq.KiloHertz, 0, freq.KiloHertz},
		{freq.KiloHertz, 1, freq.KiloHertz},
		{-freq.KiloHertz, 1, -freq.KiloHertz},
	}

	for _, test := range tests {
		if actual := test.f.Round(test.m); actual != test.expect {
			t.Errorf("Frequency(%v).Round(%v) = %v, want %v", test.f, test.m, actual, test.expect)
		}
	}
}

func TestFrequency_Abs(t *testing.T) {
	tests := []struct {
		f, expect freq.Frequency
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{-1 << 63, 1<<63 - 1},
	}

	for _, test := range tests {
		if actual := test.f.Abs(); actual != test.expect {
			t.Errorf("Frequency(%v).Abs() = %v, want %v", test.f, actual, test.expect)
		}
	}
}

func TestFromPeriod(t *testing.T) {
	tests := []struct {
		p       time.Duration
		expectF freq.Frequency
	}{
		{1 * time.Second, freq.Hertz},
		{1 * time.Millisecond, freq.KiloHertz},
		{1 * time.Microsecond, freq.MegaHertz},
		{1, freq.GigaHertz},
	}

	for _, test := range tests {
		if actual := freq.FromPeriod(test.p); actual != test.expectF {
			t.Errorf("FromPeriod(%v) = %v, want %v", test.p, actual, test.expectF)
		}
	}
}

func TestFrequency_Period(t *testing.T) {
	tests := []struct {
		f       freq.Frequency
		expectP time.Duration
	}{
		{freq.Hertz, 1 * time.Second},
		{freq.KiloHertz, 1 * time.Millisecond},
		{freq.MegaHertz, 1 * time.Microsecond},
		{freq.GigaHertz, 1},
	}

	for _, test := range tests {
		if actual := test.f.Period(); actual != test.expectP {
			t.Errorf("Frequency(%v).Period() = %v, want %v", test.f, actual, test.expectP)
		}
	}
}

func TestFrequency_JSONRoundTrip(t *testing.T) {
	tests := []freq.Frequency{
		0,
		1 * freq.NanoHertz,
		1100 * freq.NanoHertz,
		2200 * freq.MicroHertz,
		3300 * freq.MilliHertz,
		4*freq.KiloHertz + 5*freq.Hertz,
		4*freq.KiloHertz + 5001*freq.MilliHertz,
		5*freq.MegaHertz + 6*freq.KiloHertz,
		8*freq.MegaHertz + 1*freq.Hertz,
		2400 * freq.MegaHertz,
		8 * freq.KiloHertz,
		22500 * freq.Hertz,
		32 * freq.KiloHertz,
		44100 * freq.Hertz,
		48 * freq.KiloHertz,
		1<<63 - 1,
		-1 << 63,
	}

	for _, test := range tests {
		b, err := json.Marshal(test)
		if err != nil {
			t.Errorf("failed to marshal JSON: %v, freq = %v", err, test)
		}

		var actual freq.Frequency
		if err := json.Unmarshal(b, &actual); err != nil {
			t.Errorf("failed to unmarshal JSON: %v", err)
		}

		if test != actual {
			t.Errorf("round-trip failed: want %v, got %v", actual, test)
		}
	}
}

func TestFrequency_UnmarshalJSONError(t *testing.T) {
	var f freq.Frequency
	err := f.UnmarshalJSON([]byte("horalky"))
	if err == nil {
		t.Error("want error, got nil")
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		str         string
		expect      freq.Frequency
		expectError bool
	}{
		// simple
		{"0", 0, false},
		{"0Hz", 0, false},
		{"5Hz", 5 * freq.Hertz, false},
		{"30Hz", 30 * freq.Hertz, false},

		// sign
		{"+5Hz", 5 * freq.Hertz, false},
		{"-5Hz", -5 * freq.Hertz, false},
		{"-0", 0, false},
		{"+0", 0, false},
		{"-0Hz", 0, false},
		{"+0Hz", 0, false},

		// decimal
		{"5.0Hz", 5 * freq.Hertz, false},
		{"5.1Hz", 5*freq.Hertz + 100*freq.MilliHertz, false},
		{"5.Hz", 5 * freq.Hertz, false},
		{".5Hz", 500 * freq.MilliHertz, false},
		{"1.0Hz", 1 * freq.Hertz, false},
		{"1.00Hz", 1 * freq.Hertz, false},
		{"1.000Hz", 1 * freq.Hertz, false},
		{"1.005Hz", 1*freq.Hertz + 5*freq.MilliHertz, false},
		{"1.00500Hz", 1*freq.Hertz + 5*freq.MilliHertz, false},
		{"1.005000Hz", 1*freq.Hertz + 5*freq.MilliHertz, false},
		{"100.005Hz", 100*freq.Hertz + 5*freq.MilliHertz, false},
		{"100.00500Hz", 100*freq.Hertz + 5*freq.MilliHertz, false},
		{"100.005000Hz", 100*freq.Hertz + 5*freq.MilliHertz, false},

		// different units
		{"10nHz", 10 * freq.NanoHertz, false},
		{"11uHz", 11 * freq.MicroHertz, false},
		{"11µHz", 11 * freq.MicroHertz, false}, // U+00B5 = micro symbol
		{"11μHz", 11 * freq.MicroHertz, false}, // U+03BC = Greek letter mu
		{"12mHz", 12 * freq.MilliHertz, false},
		{"13cHz", 13 * freq.CentiHertz, false},
		{"14dHz", 14 * freq.DeciHertz, false},
		{"15Hz", 15 * freq.Hertz, false},
		{"16daHz", 16 * freq.DecaHertz, false},
		{"17hHz", 17 * freq.HectoHertz, false},
		{"18kHz", 18 * freq.KiloHertz, false},
		{"19MHz", 19 * freq.MegaHertz, false},
		{"2GHz", 2 * freq.GigaHertz, false},

		// composite
		{"1Hz2kHz3MHz", 1*freq.Hertz + 2*freq.KiloHertz + 3*freq.MegaHertz, false},
		{"20.5kHz6MHz", 20*freq.KiloHertz + 5*freq.HectoHertz + 6*freq.MegaHertz, false},

		// invalid
		{"", 0, true},
		{" ", 0, true},
		{"\t", 0, true},
		{"\n", 0, true},
		{"\r\n", 0, true},
		{"5", 0, true},
		{"+", 0, true},
		{"-", 0, true},
		{".", 0, true},
		{"Hz", 0, true},
		{".Hz", 0, true},
		{"+.Hz", 0, true},
		{"-.Hz", 0, true},
		{"+Hz", 0, true},
		{"-Hz", 0, true},
		{"1THz", 0, true},
		{"horalky", 0, true},
		{"1Horalky", 0, true},
		{"THz", 0, true},
		{"\uFFFD", 0, true},
		{"\uFFFDhoralky", 0, true},
		{"9223372036854775808Hz", 0, true},
		{"9223372036854775808", 0, true},
		{"-9223372036854775809Hz", 0, true},
		{"-9223372036854775809", 0, true},
	}

	for _, test := range tests {
		f, err := freq.Parse(test.str)
		if (err != nil) != test.expectError || f != test.expect {
			if test.expectError {
				t.Errorf("ParseFrequency(%q) = %v, %v, want %v, error", test.str, f, err, test.expect)
			} else {
				t.Errorf("ParseFrequency(%q) = %v, %v, want %v, nil", test.str, f, err, test.expect)
			}
		}
	}
}
