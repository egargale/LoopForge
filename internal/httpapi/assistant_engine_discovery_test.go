package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// A gateway publishes embeddings, rerankers and ASR models next to chat models.
// The first entry became the chat default, so a turn was sent to a model that
// cannot answer.
func TestDiscoverAssistantEngineModelsDropsNonChatModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[
			{"id":"Xiaomi/mimo-v2.5-asr"},
			{"id":"scint/Qwen/Qwen3-Embedding-0.6B"},
			{"id":"scint/Qwen/Qwen3-Reranker-8B"},
			{"id":"alibaba/qwen3.8-max","name":"Qwen 3.8 Max"},
			{"id":"moonshot/kimi-k3"}]}`))
	}))
	defer server.Close()

	models, err := discoverAssistantEngineModels(context.Background(), false, "key", server.URL)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	var ids []string
	for _, model := range models {
		ids = append(ids, model.ID)
	}
	if want := []string{"alibaba/qwen3.8-max", "moonshot/kimi-k3"}; !slices.Equal(ids, want) {
		t.Fatalf("chat models = %v, want %v", ids, want)
	}
	if models[0].Name != "Qwen 3.8 Max" {
		t.Fatalf("first model keeps its display name: %q", models[0].Name)
	}
}

// A gateway that publishes only non-chat ids still has to offer its list: an
// empty picker is worse than an odd default.
func TestDiscoverAssistantEngineModelsKeepsNonChatOnlyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"Xiaomi/mimo-v2.5-asr"}]}`))
	}))
	defer server.Close()

	models, err := discoverAssistantEngineModels(context.Background(), false, "key", server.URL)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(models) != 1 || models[0].ID != "Xiaomi/mimo-v2.5-asr" {
		t.Fatalf("fallback list = %v", models)
	}
}
