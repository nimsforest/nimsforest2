package runtime

import "testing"

func TestResolveDynamicSubject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		data    map[string]interface{}
		want    string
	}{
		{
			name:    "simple replacement",
			subject: "song.telegram.{chat_id}",
			data:    map[string]interface{}{"chat_id": "123456"},
			want:    "song.telegram.123456",
		},
		{
			name:    "multiple replacements",
			subject: "song.{platform}.{chat_id}",
			data:    map[string]interface{}{"platform": "telegram", "chat_id": "789"},
			want:    "song.telegram.789",
		},
		{
			name:    "no placeholders",
			subject: "song.telegram.direct",
			data:    map[string]interface{}{"chat_id": "123"},
			want:    "song.telegram.direct",
		},
		{
			name:    "missing field keeps placeholder",
			subject: "song.telegram.{missing_field}",
			data:    map[string]interface{}{"chat_id": "123"},
			want:    "song.telegram.{missing_field}",
		},
		{
			name:    "numeric value",
			subject: "song.telegram.{chat_id}",
			data:    map[string]interface{}{"chat_id": 123456},
			want:    "song.telegram.123456",
		},
		{
			name:    "NATS wildcard unchanged",
			subject: "song.telegram.>",
			data:    map[string]interface{}{"chat_id": "123"},
			want:    "song.telegram.>",
		},
		{
			name:    "empty data",
			subject: "song.telegram.{chat_id}",
			data:    map[string]interface{}{},
			want:    "song.telegram.{chat_id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDynamicSubject(tt.subject, tt.data)
			if got != tt.want {
				t.Errorf("resolveDynamicSubject() = %q, want %q", got, tt.want)
			}
		})
	}
}
