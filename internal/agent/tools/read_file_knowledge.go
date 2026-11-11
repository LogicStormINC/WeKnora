package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// WithKnowledgeImages adds the existing read_document scope to image reads;
// it does not grant tenant-wide resource access or sandbox access.
func (t *ReadFileTool) WithKnowledgeImages(read func(context.Context, string) ([]byte, *types.SearchResult, error)) *ReadFileTool {
	t.knowledgeImages = read
	t.updateDescription()
	return t
}

// ReadKnowledgeImage reuses document/tag authorization before reading storage.
// The caller looks up candidate chunk IDs; neither that lookup nor a guessed
// resource handle is authority to read the bytes.
func ReadKnowledgeImage(ctx context.Context, targets types.SearchTargets, chunkID, ref string,
	chunks interfaces.ChunkService, documents interfaces.KnowledgeService,
	images interfaces.KnowledgeBaseService,
) ([]byte, *types.SearchResult, error) {
	unavailable := fmt.Errorf("knowledge image is unavailable in the current scope")
	if _, ok := types.ParseResourcePath(ref); !ok || images == nil {
		return nil, nil, unavailable
	}
	chunk, err := authorizeChunkInSearchTargets(ctx, targets, chunkID, chunks, documents)
	if err != nil {
		return nil, nil, unavailable
	}
	parent, err := authorizeKnowledgeInSearchTargets(ctx, targets, chunk.KnowledgeID, documents)
	if err != nil || parent.KnowledgeBaseID != chunk.KnowledgeBaseID || parent.TenantID != chunk.TenantID {
		return nil, nil, unavailable
	}
	tenant := targets.GetTenantIDForKB(chunk.KnowledgeBaseID)
	if tenant == 0 {
		tenant, _ = types.TenantIDFromContext(ctx)
	}
	if tenant == 0 || tenant != chunk.TenantID {
		return nil, nil, unavailable
	}
	var infos []types.ImageInfo
	if json.Unmarshal([]byte(chunk.ImageInfo), &infos) != nil {
		return nil, nil, unavailable
	}
	for _, info := range infos {
		if info.URL != ref && info.OriginalURL != ref {
			continue
		}
		info.URL = ref
		selected, err := json.Marshal([]types.ImageInfo{info})
		if err != nil {
			return nil, nil, unavailable
		}
		source := &types.SearchResult{
			ID: chunk.ID, KnowledgeID: chunk.KnowledgeID, KnowledgeBaseID: chunk.KnowledgeBaseID,
			KnowledgeTitle: parent.Title, KnowledgeFilename: parent.FileName, ChunkType: chunk.ChunkType,
			Content: chunk.Content, SourceLocators: chunk.SourceLocators, ImageInfo: string(selected),
		}
		data, err := images.ReadChunkImage(types.WithExecutionTenant(ctx, chunk.TenantID), source)
		if err != nil {
			return nil, nil, err
		}
		return data, source, nil
	}
	return nil, nil, unavailable
}
