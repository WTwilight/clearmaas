package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

// ---------------------------------------------------------------------------
// ModelRequest parsing
// ---------------------------------------------------------------------------

func TestModelRequest_JSONParsing(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid model", `{"model":"gpt-4o"}`, false},
		{"model with group", `{"model":"gpt-4o","group":"vip"}`, false},
		{"empty model", `{"model":""}`, false}, // empty model is valid JSON, error handled later
		{"missing model", `{}`, false},
		{"invalid json", `not json`, true},
		{"trailing comma", `{"model":"gpt-4o",}`, false}, // go json allows trailing comma
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mr ModelRequest
			err := json.Unmarshal([]byte(tt.body), &mr)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Distribute error paths: guard clauses
// ---------------------------------------------------------------------------

func TestDistribute_ModelLimitEnabled_EmptyMap(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/v1/chat/completions")
	c.Set("token_model_limit_enabled", true)
	c.Set("token_model_limit", map[string]bool{}) // empty map

	body := `{"model":"gpt-4o"}`
	c.Request.Body = createBody(body)

	middleware := Distribute()
	middleware(c)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDistribute_ModelLimitEnabled_ModelNotInLimit(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/v1/chat/completions")
	c.Set("token_model_limit_enabled", true)
	c.Set("token_model_limit", map[string]bool{
		"gpt-4.1": true,
		"gpt-4o-mini": true,
	})

	body := `{"model":"gpt-4o"}` // gpt-4o not in the limit
	c.Request.Body = createBody(body)

	middleware := Distribute()
	middleware(c)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDistribute_ModelLimitEnabled_ModelInLimit(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/v1/chat/completions")
	c.Set("token_model_limit_enabled", true)
	c.Set("token_model_limit", map[string]bool{
		"gpt-4o":     true,
		"gpt-4.1":    true,
	})

	body := `{"model":"gpt-4o"}`
	c.Request.Body = createBody(body)
	c.Set("group", "default")

	middleware := Distribute()
	// This will call CacheGetRandomSatisfiedChannel which depends on DB state.
	// We just verify it doesn't 403 (model IS in limit).
	// The actual routing may fail with 503 if no channels exist, but that's OK.
	middleware(c)

	// If it returns 503, that means model IS in limit but no channel available.
	// That's expected behavior — we're testing that it doesn't 403.
	require.NotEqual(t, http.StatusForbidden, w.Code,
		"model in limit should not return 403")
}

func TestDistribute_ModelLimitDisabled_AllowsAllModels(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/v1/chat/completions")
	c.Set("token_model_limit_enabled", false)
	// token_model_limit is not set when limit is disabled

	body := `{"model":"any-model-name"}`
	c.Request.Body = createBody(body)
	c.Set("group", "default")

	middleware := Distribute()
	// Should not abort with 403. May abort with 503 if no channels, but not 403.
	middleware(c)
	require.NotEqual(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// Model name validation
// ---------------------------------------------------------------------------

func TestDistribute_EmptyModelName(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/v1/chat/completions")
	c.Set("token_model_limit_enabled", false)

	body := `{"model":""}`
	c.Request.Body = createBody(body)

	middleware := Distribute()
	middleware(c)

	// Empty model name → 400
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDistribute_MissingModelField(t *testing.T) {
	c, w := newTestContext(http.MethodPost, "/v1/chat/completions")
	c.Set("token_model_limit_enabled", false)

	body := `{}`
	c.Request.Body = createBody(body)

	middleware := Distribute()
	middleware(c)

	// Missing model field → 400
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Helper: body creation
// ---------------------------------------------------------------------------

func createBody(content string) *bodyReader {
	return &bodyReader{data: []byte(content)}
}

type bodyReader struct {
	data []byte
	pos  int
}

func (b *bodyReader) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, http.ErrBodyReadAfterConcluded
	}
	n := copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *bodyReader) Close() error { return nil }
