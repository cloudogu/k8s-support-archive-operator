package domain

import (
	"testing"
)

func TestCollectorType_GetConditionType(t *testing.T) {
	tests := []struct {
		name string
		c    CollectorType
		want string
	}{
		{
			name: "type log",
			c:    "Logs",
			want: "TODO",
		},
		{
			name: "type volume info",
			c:    "VolumeInfo",
			want: "VolumeInfoFetched",
		},
		{
			name: "type node info",
			c:    "NodeInfo",
			want: "NodeInfoFetched",
		},
		{
			name: "type events",
			c:    "Events",
			want: "EventsFetched",
		},
		{
			name: "anything else",
			c:    "blablabla",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.GetConditionType(); got != tt.want {
				t.Errorf("GetConditionType() = %v, want %v", got, tt.want)
			}
		})
	}
}
