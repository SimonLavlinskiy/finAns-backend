package service

import (
	"testing"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- buildTagTree tests ---

func TestBuildTagTree_Empty(t *testing.T) {
	result := buildTagTree(nil)
	assert.Empty(t, result)
}

func TestBuildTagTree_OnlyRoots(t *testing.T) {
	tags := []domain.Tag{
		{ID: 1, Name: "Food", Color: "#ff0000"},
		{ID: 2, Name: "Transport", Color: "#00ff00"},
	}
	result := buildTagTree(tags)

	require.Len(t, result, 2)
	names := []string{result[0].Name, result[1].Name}
	assert.Contains(t, names, "Food")
	assert.Contains(t, names, "Transport")
	for _, r := range result {
		assert.Empty(t, r.Children)
	}
}

func TestBuildTagTree_WithChildren(t *testing.T) {
	parentID := int64(1)
	tags := []domain.Tag{
		{ID: 1, Name: "Food", Color: "#ff0000"},
		{ID: 2, Name: "Cafe", Color: "#ff8888", ParentID: &parentID},
		{ID: 3, Name: "Grocery", Color: "#ff9999", ParentID: &parentID},
	}
	result := buildTagTree(tags)

	require.Len(t, result, 1)
	assert.Equal(t, "Food", result[0].Name)
	require.Len(t, result[0].Children, 2)

	childNames := []string{result[0].Children[0].Name, result[0].Children[1].Name}
	assert.Contains(t, childNames, "Cafe")
	assert.Contains(t, childNames, "Grocery")
}

func TestBuildTagTree_ChildHasParentIDSet(t *testing.T) {
	parentID := int64(10)
	tags := []domain.Tag{
		{ID: 10, Name: "Root", Color: "#000000"},
		{ID: 11, Name: "Child", Color: "#aaaaaa", ParentID: &parentID},
	}
	result := buildTagTree(tags)

	require.Len(t, result, 1)
	assert.Nil(t, result[0].ParentID)
	require.Len(t, result[0].Children, 1)
	require.NotNil(t, result[0].Children[0].ParentID)
	assert.Equal(t, int64(10), *result[0].Children[0].ParentID)
}

func TestBuildTagTree_MultipleRootsWithChildren(t *testing.T) {
	root1ID := int64(1)
	root2ID := int64(2)
	tags := []domain.Tag{
		{ID: 1, Name: "A", Color: "#aaa"},
		{ID: 2, Name: "B", Color: "#bbb"},
		{ID: 3, Name: "A1", Color: "#aaa1", ParentID: &root1ID},
		{ID: 4, Name: "B1", Color: "#bbb1", ParentID: &root2ID},
		{ID: 5, Name: "B2", Color: "#bbb2", ParentID: &root2ID},
	}
	result := buildTagTree(tags)

	require.Len(t, result, 2)
	for _, r := range result {
		switch r.Name {
		case "A":
			assert.Len(t, r.Children, 1)
		case "B":
			assert.Len(t, r.Children, 2)
		}
	}
}

func TestBuildTagTree_DuplicateIDsGraceful(t *testing.T) {
	// buildTagTree iterates the input slice for roots and builds byID map.
	// Duplicate IDs: both entries are appended to roots list (last wins in byID),
	// so result length equals the number of root entries in the input.
	tags := []domain.Tag{
		{ID: 1, Name: "First", Color: "#111"},
		{ID: 1, Name: "Second", Color: "#222"},
	}
	result := buildTagTree(tags)
	// Both entries have no parent so both appear as roots
	assert.Len(t, result, 2)
}

// --- tagToDTO tests ---

func TestTagToDTO_NoParent(t *testing.T) {
	tag := domain.Tag{ID: 5, Name: "Rent", Color: "#123456"}
	resp := tagToDTO(tag)

	assert.Equal(t, int64(5), resp.ID)
	assert.Equal(t, "Rent", resp.Name)
	assert.Equal(t, "#123456", resp.Color)
	assert.Nil(t, resp.ParentID)
	assert.Empty(t, resp.Children)
}

func TestTagToDTO_WithParent(t *testing.T) {
	pid := int64(99)
	tag := domain.Tag{ID: 7, Name: "Sub", Color: "#abcdef", ParentID: &pid}
	resp := tagToDTO(tag)

	require.NotNil(t, resp.ParentID)
	assert.Equal(t, int64(99), *resp.ParentID)
}

// --- isValidURL tests ---

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		raw   string
		valid bool
	}{
		{"https://example.com", true},
		{"http://localhost:8080/path", true},
		{"https://finanns.space/api/v1/transactions", true},
		{"not-a-url", false},
		{"", false},
		{"ftp://files.example.com", true},
		{"example.com", false},
		{"/relative/path", false},
	}

	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			got := isValidURL(tc.raw)
			assert.Equal(t, tc.valid, got, "URL: %q", tc.raw)
		})
	}
}

// --- TagWithParent via tagToDTO ---

func TestTagResponseStructure(t *testing.T) {
	resp := dto.TagResponse{
		ID:       1,
		Name:     "Root",
		Color:    "#000",
		Children: []dto.TagResponse{},
	}
	assert.Empty(t, resp.Children)
	assert.Nil(t, resp.ParentID)
}
