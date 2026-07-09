package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MonthlyBilling is the default GET /v1/billing payload (subscription credits).
type MonthlyBilling struct {
	MonthlyLimit       float64 `json:"monthlyLimit"`
	Used               float64 `json:"used"`
	OnDemandCap        float64 `json:"onDemandCap,omitempty"`
	BillingPeriodStart string  `json:"billingPeriodStart,omitempty"`
	BillingPeriodEnd   string  `json:"billingPeriodEnd,omitempty"`
	// Raw keeps unknown fields for forward compatibility.
	Raw map[string]json.RawMessage `json:"-"`
}

// WeeklyCredits is GET /v1/billing?format=credits.
type WeeklyCredits struct {
	CreditUsagePercent float64                    `json:"creditUsagePercent"`
	BillingPeriodEnd   string                     `json:"billingPeriodEnd,omitempty"`
	ProductUsage       json.RawMessage            `json:"productUsage,omitempty"`
	Raw                map[string]json.RawMessage `json:"-"`
}

// BillingSnapshot combines monthly + optional weekly views.
type BillingSnapshot struct {
	Monthly *MonthlyBilling `json:"monthly,omitempty"`
	Weekly  *WeeklyCredits  `json:"weekly,omitempty"`
}

// GetBilling fetches monthly billing for the given access token.
func (c *Client) GetBilling(ctx context.Context, accessToken string) (*MonthlyBilling, error) {
	status, _, raw, err := c.DoJSON(ctx, http.MethodGet, "/billing", nil, RequestOptions{
		AccessToken: accessToken,
		Accept:      "application/json",
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("upstream billing: status %d: %s", status, truncate(string(raw), 512))
	}
	return ParseMonthlyBilling(raw)
}

// GetBillingCredits fetches weekly credit usage (?format=credits).
func (c *Client) GetBillingCredits(ctx context.Context, accessToken string) (*WeeklyCredits, error) {
	status, _, raw, err := c.DoJSON(ctx, http.MethodGet, "/billing?format=credits", nil, RequestOptions{
		AccessToken: accessToken,
		Accept:      "application/json",
	})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("upstream billing credits: status %d: %s", status, truncate(string(raw), 512))
	}
	return ParseWeeklyCredits(raw)
}

// GetBillingSnapshot fetches monthly billing and best-effort weekly credits.
func (c *Client) GetBillingSnapshot(ctx context.Context, accessToken string) (*BillingSnapshot, error) {
	monthly, err := c.GetBilling(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	snap := &BillingSnapshot{Monthly: monthly}
	if weekly, werr := c.GetBillingCredits(ctx, accessToken); werr == nil {
		snap.Weekly = weekly
	}
	return snap, nil
}

// ParseMonthlyBilling parses a /billing JSON body.
//
// Supports:
//   - flat: {"monthlyLimit":4000,"used":1421,...}
//   - nested config: {"config":{...}}
//   - protobuf-ish numbers: {"monthlyLimit":{"val":20000},"used":{"val":2704}}
func ParseMonthlyBilling(raw []byte) (*MonthlyBilling, error) {
	if len(bytesTrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("upstream billing: empty body")
	}
	root := billingObject(raw)
	cfg := root
	if nested, ok := root["config"]; ok {
		if obj := billingObject(nested); len(obj) > 0 {
			cfg = obj
		}
	}

	out := &MonthlyBilling{}
	if v, ok := numberFromAny(cfg["monthlyLimit"]); ok {
		out.MonthlyLimit = v
	} else if v, ok := numberFromAny(cfg["monthly_limit"]); ok {
		out.MonthlyLimit = v
	}
	if v, ok := numberFromAny(cfg["used"]); ok {
		out.Used = v
	}
	if v, ok := numberFromAny(cfg["onDemandCap"]); ok {
		out.OnDemandCap = v
	} else if v, ok := numberFromAny(cfg["on_demand_cap"]); ok {
		out.OnDemandCap = v
	}
	out.BillingPeriodStart = stringFromAny(cfg["billingPeriodStart"])
	if out.BillingPeriodStart == "" {
		out.BillingPeriodStart = stringFromAny(cfg["billing_period_start"])
	}
	out.BillingPeriodEnd = stringFromAny(cfg["billingPeriodEnd"])
	if out.BillingPeriodEnd == "" {
		out.BillingPeriodEnd = stringFromAny(cfg["billing_period_end"])
	}
	return out, nil
}

// ParseWeeklyCredits parses a /billing?format=credits body.
//
// Supports flat fields and nested {"config":{...}} with optional {"val":N} wrappers.
func ParseWeeklyCredits(raw []byte) (*WeeklyCredits, error) {
	if len(bytesTrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("upstream billing credits: empty body")
	}
	root := billingObject(raw)
	cfg := root
	if nested, ok := root["config"]; ok {
		if obj := billingObject(nested); len(obj) > 0 {
			cfg = obj
		}
	}

	out := &WeeklyCredits{}
	if v, ok := numberFromAny(cfg["creditUsagePercent"]); ok {
		out.CreditUsagePercent = v
	} else if v, ok := numberFromAny(cfg["credit_usage_percent"]); ok {
		out.CreditUsagePercent = v
	}
	out.BillingPeriodEnd = stringFromAny(cfg["billingPeriodEnd"])
	if out.BillingPeriodEnd == "" {
		out.BillingPeriodEnd = stringFromAny(cfg["billing_period_end"])
	}
	if pu, ok := cfg["productUsage"]; ok && len(pu) > 0 {
		out.ProductUsage = append(json.RawMessage(nil), pu...)
	} else if pu, ok := cfg["product_usage"]; ok && len(pu) > 0 {
		out.ProductUsage = append(json.RawMessage(nil), pu...)
	}
	return out, nil
}

func billingObject(raw []byte) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// numberFromAny accepts JSON number or {"val": number|string}.
func numberFromAny(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		var parsed float64
		if _, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &parsed); err == nil {
			return parsed, true
		}
	}
	var wrap struct {
		Val json.RawMessage `json:"val"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && len(wrap.Val) > 0 {
		return numberFromAny(wrap.Val)
	}
	return 0, false
}

func stringFromAny(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return ""
}

func bytesTrimSpace(b []byte) []byte {
	return bytes.TrimSpace(b)
}

// RemainingCredits returns monthlyLimit - used (floored at 0).
func (m *MonthlyBilling) RemainingCredits() float64 {
	if m == nil {
		return 0
	}
	rem := m.MonthlyLimit - m.Used
	if rem < 0 {
		return 0
	}
	return rem
}

// UsagePercent returns used/limit * 100. Zero limit → 0.
func (m *MonthlyBilling) UsagePercent() float64 {
	if m == nil || m.MonthlyLimit <= 0 {
		return 0
	}
	return (m.Used / m.MonthlyLimit) * 100
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
