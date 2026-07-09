package upstream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	ApplyHeaders(req, HeaderInput{
		AccessToken:      "tok",
		Model:            "grok-4.5",
		ConvID:           "conv-1",
		ClientVersion:    "0.2.93",
		ClientIdentifier: "grok-pager",
		TokenAuth:        "xai-grok-cli",
		UserAgent:        "grok-pager/0.2.93",
		Accept:           "application/json",
	})
	checks := map[string]string{
		"Authorization":            "Bearer tok",
		"X-XAI-Token-Auth":         "xai-grok-cli",
		"x-grok-client-version":    "0.2.93",
		"x-grok-client-identifier": "grok-pager",
		"x-grok-model-override":    "grok-4.5",
		"x-grok-conv-id":           "conv-1",
		"User-Agent":               "grok-pager/0.2.93",
		"Accept":                   "application/json",
	}
	for k, want := range checks {
		if got := req.Header.Get(k); got != want {
			t.Errorf("%s: got %q want %q", k, got, want)
		}
	}
}

func TestApplyHeaders_ExtraOverrides(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	extra := http.Header{}
	extra.Set("User-Agent", "custom-ua")
	extra.Set("X-Custom", "1")
	ApplyHeaders(req, HeaderInput{
		AccessToken: "t",
		UserAgent:   "default-ua",
		Extra:       extra,
	})
	if got := req.Header.Get("User-Agent"); got != "custom-ua" {
		t.Errorf("ua=%q", got)
	}
	if got := req.Header.Get("X-Custom"); got != "1" {
		t.Errorf("custom=%q", got)
	}
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" && r.URL.Path != "/models" {
			// Client joins base+path; base includes /v1 so path is /models → /v1/models
			if !strings.HasSuffix(r.URL.Path, "/models") {
				t.Errorf("path=%s", r.URL.Path)
			}
		}
		assertGrokHeaders(t, r, "")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{
					"id":                        "grok-4.5",
					"api_backend":               "responses",
					"context_window":            500000,
					"supports_reasoning_effort": true,
				},
				{
					"id":             "grok-composer-2.5-fast",
					"api_backend":    "responses",
					"context_window": 200000,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})
	list, err := c.ListModels(context.Background(), "access-tok")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 2 {
		t.Fatalf("len=%d", len(list.Data))
	}
	if list.Find("grok-4.5") == nil {
		t.Fatal("missing grok-4.5")
	}
	ids := list.IDs()
	if len(ids) != 2 {
		t.Fatalf("ids=%v", ids)
	}
}

func TestParseModelList_CacheShape(t *testing.T) {
	raw := []byte(`{
  "models": {
    "grok-4.5": {
      "info": {
        "id": "grok-4.5",
        "api_backend": "responses",
        "context_window": 500000
      }
    }
  }
}`)
	list, err := ParseModelList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if list.Find("grok-4.5") == nil {
		t.Fatal("not found")
	}
}

func TestGetBilling(t *testing.T) {
	var sawCredits bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertGrokHeaders(t, r, "")
		if strings.Contains(r.URL.RawQuery, "format=credits") {
			sawCredits = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"creditUsagePercent": 36.0,
				"billingPeriodEnd":   "2026-07-01T00:00:00Z",
			})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/billing") {
			t.Errorf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"monthlyLimit":     4000,
			"used":             1421,
			"onDemandCap":      0,
			"billingPeriodEnd": "2026-08-01T00:00:00Z",
		})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})
	m, err := c.GetBilling(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if m.MonthlyLimit != 4000 || m.Used != 1421 {
		t.Fatalf("%+v", m)
	}
	if m.RemainingCredits() != 4000-1421 {
		t.Errorf("remaining=%v", m.RemainingCredits())
	}
	snap, err := c.GetBillingSnapshot(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Monthly == nil || snap.Weekly == nil {
		t.Fatalf("snap=%+v sawCredits=%v", snap, sawCredits)
	}
	if snap.Weekly.CreditUsagePercent != 36 {
		t.Errorf("weekly=%+v", snap.Weekly)
	}
}

func TestParseMonthlyBilling_NestedConfig(t *testing.T) {
	raw := []byte(`{"config":{"monthlyLimit":100,"used":10,"billingPeriodEnd":"2026-01-01T00:00:00Z"}}`)
	m, err := ParseMonthlyBilling(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.MonthlyLimit != 100 || m.Used != 10 {
		t.Fatalf("%+v", m)
	}
}

func TestParseMonthlyBilling_ConfigValWrapper(t *testing.T) {
	// Live cli-chat-proxy shape (2026-07): numbers wrapped as {"val":N} under config.
	raw := []byte(`{"config":{"monthlyLimit":{"val":20000},"used":{"val":2704},"onDemandCap":{"val":0},"billingPeriodStart":"2026-07-01T00:00:00+00:00","billingPeriodEnd":"2026-08-01T00:00:00+00:00"}}`)
	m, err := ParseMonthlyBilling(raw)
	if err != nil {
		t.Fatal(err)
	}
	if m.MonthlyLimit != 20000 || m.Used != 2704 {
		t.Fatalf("limit/used: %+v", m)
	}
	if m.BillingPeriodStart == "" || m.BillingPeriodEnd == "" {
		t.Fatalf("period missing: %+v", m)
	}
	if m.RemainingCredits() != 20000-2704 {
		t.Errorf("remaining=%v", m.RemainingCredits())
	}
}

func TestParseWeeklyCredits_ConfigValWrapper(t *testing.T) {
	raw := []byte(`{"config":{"creditUsagePercent":66.0,"billingPeriodEnd":"2026-07-13T04:20:32.192591+00:00","productUsage":[{"product":"Api","usagePercent":66.0}]}}`)
	w, err := ParseWeeklyCredits(raw)
	if err != nil {
		t.Fatal(err)
	}
	if w.CreditUsagePercent != 66 {
		t.Fatalf("percent=%v", w.CreditUsagePercent)
	}
	if w.BillingPeriodEnd == "" {
		t.Fatal("missing period end")
	}
	if len(w.ProductUsage) == 0 {
		t.Fatal("expected productUsage")
	}
}

func TestPostResponses_DoesNotConsumeBody(t *testing.T) {
	const payload = `data: {"type":"response.output_text.delta","delta":"hi"}` + "\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method=%s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/responses") {
			t.Errorf("path=%s", r.URL.Path)
		}
		assertGrokHeaders(t, r, "grok-4.5")
		if r.Header.Get("x-grok-conv-id") != "c1" {
			t.Errorf("conv=%q", r.Header.Get("x-grok-conv-id"))
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("accept=%q", r.Header.Get("Accept"))
		}
		body, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		if m["model"] != "grok-4.5" {
			t.Errorf("body model=%v", m["model"])
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, payload)
	}))
	t.Cleanup(srv.Close)

	c := NewClient(Config{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})
	resp, err := c.PostResponses(context.Background(), map[string]any{
		"model":  "grok-4.5",
		"input":  "hello",
		"stream": true,
	}, PostResponsesOptions{
		AccessToken: "tok",
		Model:       "grok-4.5",
		ConvID:      "c1",
		Stream:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Critical: body not pre-consumed — we can still read the stream.
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != payload {
		t.Fatalf("body=%q", got)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPostResponsesJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertGrokHeaders(t, r, "grok-4.5")
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("accept=%q", r.Header.Get("Accept"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_1",
			"object": "response",
			"status": "completed",
		})
	}))
	t.Cleanup(srv.Close)
	c := NewClient(Config{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})
	status, _, raw, err := c.PostResponsesJSON(context.Background(), []byte(`{"model":"grok-4.5","input":"x"}`), PostResponsesOptions{
		AccessToken: "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Fatalf("status=%d", status)
	}
	if !strings.Contains(string(raw), "resp_1") {
		t.Fatalf("raw=%s", raw)
	}
}

func TestPostResponses_ModelFromBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-grok-model-override") != "grok-composer-2.5-fast" {
			t.Errorf("model override=%q", r.Header.Get("x-grok-model-override"))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient(Config{BaseURL: srv.URL + "/v1", HTTPClient: srv.Client()})
	resp, err := c.PostResponses(context.Background(), json.RawMessage(`{"model":"grok-composer-2.5-fast","input":[]}`), PostResponsesOptions{
		AccessToken: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(Config{})
	if c.BaseURL() != DefaultBaseURL {
		t.Errorf("base=%s", c.BaseURL())
	}
	cfg := c.Config()
	if cfg.TokenAuth != DefaultTokenAuth || cfg.ClientVersion != DefaultClientVersion {
		t.Errorf("%+v", cfg)
	}
}

func TestJoinURL(t *testing.T) {
	if got := joinURL("https://cli-chat-proxy.grok.com/v1/", "/models"); got != "https://cli-chat-proxy.grok.com/v1/models" {
		t.Fatal(got)
	}
	if got := joinURL("https://cli-chat-proxy.grok.com/v1", "billing"); got != "https://cli-chat-proxy.grok.com/v1/billing" {
		t.Fatal(got)
	}
}

func assertGrokHeaders(t *testing.T, r *http.Request, model string) {
	t.Helper()
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		t.Errorf("Authorization=%q", r.Header.Get("Authorization"))
	}
	if r.Header.Get("X-XAI-Token-Auth") != DefaultTokenAuth {
		t.Errorf("X-XAI-Token-Auth=%q", r.Header.Get("X-XAI-Token-Auth"))
	}
	if r.Header.Get("x-grok-client-version") == "" {
		t.Error("missing x-grok-client-version")
	}
	if r.Header.Get("x-grok-client-identifier") == "" {
		t.Error("missing x-grok-client-identifier")
	}
	if r.Header.Get("User-Agent") == "" {
		t.Error("missing User-Agent")
	}
	if model != "" && r.Header.Get("x-grok-model-override") != model {
		t.Errorf("model-override=%q want %q", r.Header.Get("x-grok-model-override"), model)
	}
}
