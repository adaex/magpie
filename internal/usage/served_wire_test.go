package usage

import "testing"

// A model listed by an id of the user's own goes out under a provider's
// deployment id (a Volcengine Ark ep-… endpoint), and its reply names the
// model's dated version: the deployment id can never equal it, but magpie's
// own name for the model does.
func TestSwappedOnWire(t *testing.T) {
	for _, c := range []struct {
		name                string
		sent, model, served string
		want                bool
	}{
		{"endpoint id, dated reply of the same model", "ep-20260101000000-abcde", "deepseek-v4.1-flash", "deepseek-v4-1-flash-261017", false},
		{"endpoint id, latest-tail reply of the same model", "ep-20260101000000-abcde", "doubao-seed-evolving", "doubao-seed-evolving-latest", false},
		{"endpoint id, another model answered", "ep-20260101000000-abcde", "deepseek-v4.1-flash", "kimi-k3.1-pro-261105", true},
		{"bare id, dated reply of the same model", "deepseek-v4.1-flash", "deepseek-v4.1-flash", "deepseek-v4-1-flash-261017", false},
		{"bare id, another model answered", "deepseek-v4.1-flash", "deepseek-v4.1-flash", "kimi-k3.1-pro", true},
		{"no reply model named", "ep-20260101000000-abcde", "deepseek-v4.1-flash", "", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := SwappedOnWire(c.sent, c.model, c.served); got != c.want {
				t.Errorf("SwappedOnWire(%q, %q, %q) = %v, want %v", c.sent, c.model, c.served, got, c.want)
			}
		})
	}
}
