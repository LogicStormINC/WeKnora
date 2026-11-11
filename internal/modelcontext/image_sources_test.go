package modelcontext

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestImageSourcesUseOnlyKnownHandlesAndAccountForUserImages(t *testing.T) {
	registry := NewRegistry(true)
	registry.RegisterChunk(ChunkReference{ChunkID: "known-source"})
	note := registry.ImageSourcesNote([]string{"known-source", "unverified-source"}, 2)
	require.Contains(t, note, "Image 3: chunk chunk://1.")
	require.Contains(t, note, "Image 4: source unavailable")
	require.NotContains(t, note, "unverified-source")
	require.NotContains(t, note, "known-source")
	require.Empty(t, registry.ChunkHandle("unverified-source"))
	stored := &types.ToolResult{Data: map[string]interface{}{"context_images": []string{"known-source"}}}
	require.Empty(t, registry.ToolImageSourcesNote(stored), "stored history has no image bytes")
	var missing *Registry
	require.Contains(t, missing.ImageSourcesNote([]string{"known-source"}, 0), "source unavailable")
}

func TestReadImageRegistersItsOwnCitationInsteadOfParent(t *testing.T) {
	registry := NewRegistry(true)
	result := &types.ToolResult{Success: true, Output: "image bytes attached", Images: []string{"data:image/png;base64,eA=="}, Data: map[string]interface{}{
		"display_type": "image_view", "context_images": []string{"image-page-6"},
		"results": []map[string]interface{}{{"chunk_id": "image-page-6", "knowledge_id": "report", "knowledge_base_id": "kb", "knowledge_title": "report.pdf", "chunk_type": "image_vector", "content": "Original page image"}},
	}}
	output := registry.ModelToolResultForTool("read_file", result)
	require.Contains(t, output, "chunk://1")
	require.Contains(t, output, "image bytes attached")
	require.Equal(t, "chunk://1", registry.ChunkHandle("image-page-6"))
	require.Contains(t, registry.ToolImageSourcesNote(result), "Image 1: chunk chunk://1")
}
