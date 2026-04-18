package fmt

import (
	"testing"
)

// TestFormatStable is the W18-P03-T01 Verify target. Format is
// idempotent — formatting a formatted source is a no-op — and the
// normaliser covers CRLF, tabs, trailing whitespace, and excess
// blank lines.
func TestFormatStable(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"crlf-to-lf",
			"fn main() -> I32 {\r\n    return 0;\r\n}\r\n",
			"fn main() -> I32 {\n    return 0;\n}\n",
		},
		{
			"trim-trailing-ws",
			"fn main() -> I32 {   \n    return 0;  \n}\n",
			"fn main() -> I32 {\n    return 0;\n}\n",
		},
		{
			"collapse-blank-runs",
			"fn a() {}\n\n\n\nfn b() {}\n",
			"fn a() {}\n\nfn b() {}\n",
		},
		{
			"expand-tabs",
			"fn main() {\n\treturn 0;\n}\n",
			"fn main() {\n    return 0;\n}\n",
		},
		{
			"idempotent",
			"fn main() -> I32 {\n    return 42;\n}\n",
			"fn main() -> I32 {\n    return 42;\n}\n",
		},
		{
			"no-leading-blanks",
			"\n\n\nfn main() {}\n",
			"fn main() {}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(Format([]byte(tc.in)))
			if got != tc.want {
				t.Errorf("Format mismatch:\n--- got ---\n%q\n--- want ---\n%q", got, tc.want)
			}
			// Idempotency: running Format on the result must
			// produce the same bytes.
			again := string(Format([]byte(got)))
			if again != got {
				t.Errorf("Format not idempotent:\n--- first ---\n%q\n--- second ---\n%q", got, again)
			}
		})
	}
}
