package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
)

// arkEndpoint is a Volcengine Ark-style custom provider: the model is
// listed by its bare id but requested through a custom inference endpoint
// id (ep-…), and the reply names the model's dated version with the
// version's dots written as hyphens (deepseek-v4.1-flash answered as
// deepseek-v4-1-flash-261017).
type arkEndpoint struct {
	served string
	got    string
}

func (a *arkEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	a.got = modelOf(b)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"id":"x","object":"chat.completion","model":"`+a.served+
		`","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],`+
		`"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
}

// A model listed by an id of the user's own and served through a
// deployment id is not swapped when its reply names its dated version:
// the deployment id can never equal a model version, but magpie's own
// name for the model does.
func TestEndpointIDServedByDatedModelNameIsNoSwap(t *testing.T) {
	fresh(t)
	up := &arkEndpoint{served: "deepseek-v4-1-flash-261017"}
	srv := httptest.NewServer(up)
	t.Cleanup(srv.Close)

	if err := provider.Save(provider.Provider{ID: "ark", Name: "Ark", Key: "k",
		Chat: srv.URL + "/v1", Models: []string{"deepseek-v4.1-flash"}}); err != nil {
		t.Fatal(err)
	}
	s := settings.Load()
	s.ModelIDs = map[string]string{"ark/deepseek-v4.1-flash": "ep-20260101000000-abcde"}
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}

	gw := New()
	rec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(
		`{"model":"ark/deepseek-v4.1-flash","messages":[{"role":"user","content":"hi"}]}`)))
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	if up.got != "ep-20260101000000-abcde" {
		t.Fatalf("request went out as %q, want the endpoint id", up.got)
	}
	gw.trace.mu.Lock()
	r := *gw.trace.routes[len(gw.trace.routes)-1]
	gw.trace.mu.Unlock()
	if r.Swapped || len(r.Tries) == 0 || r.Tries[len(r.Tries)-1].Swapped {
		b, _ := json.Marshal(r)
		t.Errorf("dated reply to an endpoint id marked swapped: %s", b)
	}
	rows, _, _ := usage.Ledger(usage.All, usage.Filter{})
	for _, row := range rows {
		if row.Swapped {
			t.Errorf("usage row marked swapped: model %q served %q", row.Model, row.Served)
		}
	}
}

// A reply naming another model altogether is still a swap behind an
// endpoint id: the internal name catches it even though the deployment
// id matches nothing the vendor names.
func TestEndpointIDAnsweredByAnotherModelIsSwap(t *testing.T) {
	fresh(t)
	up := &arkEndpoint{served: "kimi-k3.1-pro-261105"}
	srv := httptest.NewServer(up)
	t.Cleanup(srv.Close)

	if err := provider.Save(provider.Provider{ID: "ark", Name: "Ark", Key: "k",
		Chat: srv.URL + "/v1", Models: []string{"deepseek-v4.1-flash"}}); err != nil {
		t.Fatal(err)
	}
	s := settings.Load()
	s.ModelIDs = map[string]string{"ark/deepseek-v4.1-flash": "ep-20260101000000-abcde"}
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}

	gw := New()
	rec := httptest.NewRecorder()
	gw.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(
		`{"model":"ark/deepseek-v4.1-flash","messages":[{"role":"user","content":"hi"}]}`)))
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	gw.trace.mu.Lock()
	r := *gw.trace.routes[len(gw.trace.routes)-1]
	gw.trace.mu.Unlock()
	if !r.Swapped || len(r.Tries) == 0 || !r.Tries[len(r.Tries)-1].Swapped {
		b, _ := json.Marshal(r)
		t.Errorf("another model answering an endpoint id not marked swapped: %s", b)
	}
}
