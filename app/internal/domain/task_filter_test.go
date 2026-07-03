package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskFilter_Normalize(t *testing.T) {
	tests := []struct {
		name         string
		filter       TaskFilter
		wantPage     int
		wantPageSize int
	}{
		{"defaults applied", TaskFilter{TeamID: 1}, 1, 20},
		{"oversized page size clamped", TaskFilter{TeamID: 1, Page: 2, PageSize: 1000}, 2, 100},
		{"valid values kept", TaskFilter{TeamID: 1, Page: 3, PageSize: 50}, 3, 50},
		{"negative values replaced", TaskFilter{TeamID: 1, Page: -1, PageSize: -5}, 1, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := tt.filter.Normalize(20, 100)
			assert.Equal(t, tt.wantPage, normalized.Page)
			assert.Equal(t, tt.wantPageSize, normalized.PageSize)
		})
	}
}

func TestTaskFilter_Offset(t *testing.T) {
	assert.Equal(t, 0, TaskFilter{Page: 1, PageSize: 20}.Offset())
	assert.Equal(t, 40, TaskFilter{Page: 3, PageSize: 20}.Offset())
}
