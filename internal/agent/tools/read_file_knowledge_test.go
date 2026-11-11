package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type knowledgeImageTestChunks struct {
	interfaces.ChunkService
	chunk *types.Chunk
}

func (s *knowledgeImageTestChunks) GetChunkByIDOnly(context.Context, string) (*types.Chunk, error) {
	return s.chunk, nil
}

type knowledgeImageTestStorage struct {
	interfaces.KnowledgeBaseService
	reads int
	data  []byte
	ref   string
}

func (s *knowledgeImageTestStorage) ReadChunkImage(_ context.Context, row *types.SearchResult) ([]byte, error) {
	s.reads++
	var infos []types.ImageInfo
	if err := json.Unmarshal([]byte(row.ImageInfo), &infos); err != nil {
		return nil, err
	}
	s.ref = infos[0].URL
	return s.data, nil
}

func TestReadFileKnowledgeImagesKeepDocumentAndTenantScope(t *testing.T) {
	const ref = "resource://abcdefghijklmnopqrstuv"
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	storage := &knowledgeImageTestStorage{data: data.Bytes()}
	chunk := &types.Chunk{ID: "chunk", TenantID: 7, KnowledgeID: "doc", KnowledgeBaseID: "kb", IsEnabled: true,
		ImageInfo: `[{"url":"resource://anotherImageReference1"},{"url":"` + ref + `"}]`}
	chunks := &knowledgeImageTestChunks{chunk: chunk}
	docs := &readDocKnowledgeService{docs: map[string]*types.Knowledge{"doc": {ID: "doc", TenantID: 7, KnowledgeBaseID: "kb"}}}
	targets := types.SearchTargets{{Type: types.SearchTargetTypeKnowledge, KnowledgeBaseID: "kb", TenantID: 7, KnowledgeIDs: []string{"doc"}}}
	reader := NewReadFileTool(nil).WithKnowledgeImages(func(ctx context.Context, path string) ([]byte, *types.SearchResult, error) {
		return ReadKnowledgeImage(ctx, targets, chunk.ID, path, chunks, docs, storage)
	})
	read := func(path string) *types.ToolResult {
		args, err := json.Marshal(ReadFileInput{Path: path, Detail: "full", Focus: "arrow direction"})
		require.NoError(t, err)
		result, err := reader.Execute(t.Context(), args)
		require.NoError(t, err)
		return result
	}
	got := read(ref)
	require.True(t, got.Success, got.Error)
	require.Len(t, got.Images, 1)
	require.Equal(t, []string{"chunk"}, got.Data["context_images"])
	rows := got.Data["results"].([]map[string]interface{})
	require.Equal(t, "doc", rows[0]["knowledge_id"])
	require.Equal(t, "kb", rows[0]["knowledge_base_id"])
	require.Equal(t, "full", got.ImageDetail)
	require.Equal(t, ref, storage.ref, "read the requested image, not the chunk's first image")
	for name, change := range map[string]func(){
		"different document": func() { targets[0].KnowledgeIDs = []string{"other"} },
		"different tenant":   func() { targets[0].TenantID = 8 },
		"disabled chunk":     func() { chunk.IsEnabled = false },
		"deleted parent":     func() { delete(docs.docs, "doc") },
	} {
		t.Run(name, func(t *testing.T) {
			targets[0].KnowledgeIDs = []string{"doc"}
			targets[0].TenantID = 7
			chunk.IsEnabled = true
			docs.docs["doc"] = &types.Knowledge{ID: "doc", TenantID: 7, KnowledgeBaseID: "kb"}
			change()
			before := storage.reads
			require.False(t, read(ref).Success)
			require.Equal(t, before, storage.reads, "denied resources must never reach storage")
		})
	}
}
