package tui

import (
	"strings"
	"testing"
)

func TestRenderLatencyBreakdown(t *testing.T) {
	tests := []struct {
		name         string
		modelStats   map[string]any
		wantEmpty    bool
		wantContains string
	}{
		{
			name:       "no details",
			modelStats: map[string]any{},
			wantEmpty:  true,
		},
		{
			name: "empty details",
			modelStats: map[string]any{
				"details": []any{},
			},
			wantEmpty: true,
		},
		{
			name: "details with zero latency",
			modelStats: map[string]any{
				"details": []any{
					map[string]any{
						"latency_ms": float64(0),
					},
				},
			},
			wantEmpty: true,
		},
		{
			name: "single request with latency",
			modelStats: map[string]any{
				"details": []any{
					map[string]any{
						"latency_ms": float64(1500),
					},
				},
			},
			wantEmpty:    false,
			wantContains: "avg 1500ms  min 1500ms  max 1500ms",
		},
		{
			name: "multiple requests with varying latency",
			modelStats: map[string]any{
				"details": []any{
					map[string]any{
						"latency_ms": float64(100),
					},
					map[string]any{
						"latency_ms": float64(200),
					},
					map[string]any{
						"latency_ms": float64(300),
					},
				},
			},
			wantEmpty:    false,
			wantContains: "avg 200ms  min 100ms  max 300ms",
		},
		{
			name: "mixed valid and invalid latency values",
			modelStats: map[string]any{
				"details": []any{
					map[string]any{
						"latency_ms": float64(500),
					},
					map[string]any{
						"latency_ms": float64(0),
					},
					map[string]any{
						"latency_ms": float64(1500),
					},
				},
			},
			wantEmpty:    false,
			wantContains: "avg 1000ms  min 500ms  max 1500ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := usageTabModel{}
			result := m.renderLatencyBreakdown(tt.modelStats)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("renderLatencyBreakdown() = %q, want empty string", result)
				}
				return
			}

			if result == "" {
				t.Errorf("renderLatencyBreakdown() = empty, want non-empty string")
				return
			}

			if tt.wantContains != "" && !strings.Contains(result, tt.wantContains) {
				t.Errorf("renderLatencyBreakdown() = %q, want to contain %q", result, tt.wantContains)
			}
		})
	}
}

func TestUsageTimeTranslations(t *testing.T) {
	prevLocale := CurrentLocale()
	t.Cleanup(func() {
		SetLocale(prevLocale)
	})

	tests := []struct {
		locale string
		want   string
	}{
		{locale: "en", want: "Time"},
		{locale: "zh", want: "时间"},
	}

	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			SetLocale(tt.locale)
			if got := T("usage_time"); got != tt.want {
				t.Fatalf("T(usage_time) = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHourlyStatsFromUsagePrefersTopLevelBuckets(t *testing.T) {
	usageMap := map[string]any{
		"requests_by_hour": map[string]any{"09": float64(2)},
		"apis": map[string]any{
			"key": map[string]any{
				"models": map[string]any{
					"model": map[string]any{
						"details": []any{
							map[string]any{"timestamp": "2026-03-20T10:15:00Z"},
						},
					},
				},
			},
		},
	}

	got := hourlyStatsFromUsage(usageMap, "requests_by_hour", "total_requests")
	if got["09"] != float64(2) {
		t.Fatalf("top-level bucket 09 = %v, want 2", got["09"])
	}
	if _, ok := got["10"]; ok {
		t.Fatalf("derived bucket 10 present despite non-empty top-level buckets: %#v", got)
	}
}

func TestHourlyStatsFromUsageDerivesFromImportedDetails(t *testing.T) {
	usageMap := map[string]any{
		"requests_by_hour": map[string]any{},
		"tokens_by_hour":   map[string]any{},
		"apis": map[string]any{
			"key": map[string]any{
				"models": map[string]any{
					"model-a": map[string]any{
						"details": []any{
							map[string]any{
								"timestamp": "2026-03-20T12:15:00Z",
								"tokens":    map[string]any{"total_tokens": float64(30)},
							},
							map[string]any{
								"timestamp": "2026-03-20T12:45:00Z",
								"tokens":    map[string]any{"total_tokens": float64(20)},
							},
							map[string]any{
								"timestamp": "2026-03-20T13:00:00Z",
								"tokens":    map[string]any{"total_tokens": float64(5)},
							},
						},
					},
				},
			},
		},
	}

	requests := hourlyStatsFromUsage(usageMap, "requests_by_hour", "total_requests")
	if requests["12"] != float64(2) || requests["13"] != float64(1) {
		t.Fatalf("derived request buckets = %#v, want 12:2 and 13:1", requests)
	}

	tokens := hourlyStatsFromUsage(usageMap, "tokens_by_hour", "total_tokens")
	if tokens["12"] != float64(50) || tokens["13"] != float64(5) {
		t.Fatalf("derived token buckets = %#v, want 12:50 and 13:5", tokens)
	}
}

func TestUsageTabRendersHourlyChartsFromImportedDetails(t *testing.T) {
	m := usageTabModel{
		width:  80,
		height: 24,
		usage: map[string]any{
			"usage": map[string]any{
				"total_requests":   float64(1),
				"total_tokens":     float64(30),
				"requests_by_hour": map[string]any{},
				"tokens_by_hour":   map[string]any{},
				"apis": map[string]any{
					"key": map[string]any{
						"total_requests": float64(1),
						"total_tokens":   float64(30),
						"models": map[string]any{
							"model-a": map[string]any{
								"total_requests": float64(1),
								"total_tokens":   float64(30),
								"details": []any{
									map[string]any{
										"timestamp": "2026-03-20T12:15:00Z",
										"tokens":    map[string]any{"total_tokens": float64(30)},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	content := m.renderContent()
	if !strings.Contains(content, T("usage_req_by_hour")) {
		t.Fatalf("rendered content missing requests-by-hour chart: %q", content)
	}
	if !strings.Contains(content, T("usage_tok_by_hour")) {
		t.Fatalf("rendered content missing tokens-by-hour chart: %q", content)
	}
	if !strings.Contains(content, "12") {
		t.Fatalf("rendered content missing derived hour label 12: %q", content)
	}
}
