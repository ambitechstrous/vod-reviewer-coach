package handlers

import (
	"encoding/json"
	"testing"
)

func TestExtractPayloads(t *testing.T) {
	t.Run("flat payload falls back to a single message", func(t *testing.T) {
		raw := json.RawMessage(`{"user_id":"u1","video_key":"v1","prompt":"test"}`)

		got := extractPayloads(raw)

		if len(got) != 1 || string(got[0]) != string(raw) {
			t.Fatalf("extractPayloads(%s) = %s, want a single message equal to the input", raw, got)
		}
	})

	t.Run("SQS envelope unwraps each record's body", func(t *testing.T) {
		raw := json.RawMessage(`{"Records":[
			{"body":"{\"user_id\":\"u1\",\"video_key\":\"v1\",\"prompt\":\"a\"}"},
			{"body":"{\"user_id\":\"u2\",\"video_key\":\"v2\",\"prompt\":\"b\"}"}
		]}`)

		got := extractPayloads(raw)

		if len(got) != 2 {
			t.Fatalf("extractPayloads returned %d messages, want 2", len(got))
		}

		var first AnalyzerEvent
		if err := json.Unmarshal(got[0], &first); err != nil {
			t.Fatalf("unmarshal first message: %v", err)
		}
		if first.UserID != "u1" || first.VideoID != "v1" {
			t.Errorf("first message = %+v, want UserID=u1 VideoID=v1", first)
		}

		var second AnalyzerEvent
		if err := json.Unmarshal(got[1], &second); err != nil {
			t.Fatalf("unmarshal second message: %v", err)
		}
		if second.UserID != "u2" || second.VideoID != "v2" {
			t.Errorf("second message = %+v, want UserID=u2 VideoID=v2", second)
		}
	})
}
