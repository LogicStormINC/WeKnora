package service

import (
	"context"
	"fmt"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/types"
)

func (s *agentService) registerKnowledgeImageFiles(registry *tools.ToolRegistry, config *types.AgentConfig) {
	if s.db == nil || len(config.SearchTargets) == 0 {
		return
	}
	// Keep the existing tool admission: image reads do not create a new KB grant.
	if _, err := registry.GetTool(tools.ToolReadDocument); err != nil {
		return
	}
	var reader *tools.ReadFileTool
	if existing, err := registry.GetTool(tools.ToolReadFile); err == nil {
		reader, _ = existing.(*tools.ReadFileTool)
	}
	if reader == nil {
		reader = tools.NewReadFileTool(nil)
		registry.RegisterTool(reader)
	}
	reader.WithKnowledgeImages(func(ctx context.Context, ref string) ([]byte, *types.SearchResult, error) {
		ids, err := knowledgeImageChunkIDs(ctx, s.db, config.SearchTargets, ref)
		if err != nil {
			return nil, nil, err
		}
		for _, id := range ids {
			data, chunk, err := tools.ReadKnowledgeImage(ctx, config.SearchTargets, id, ref,
				s.chunkService, s.knowledgeService, s.knowledgeBaseService)
			if err == nil {
				return data, chunk, nil
			}
		}
		return nil, nil, fmt.Errorf("knowledge image is unavailable in the current scope")
	})
}

func knowledgeImageChunkIDs(ctx context.Context, db *gorm.DB, targets types.SearchTargets, ref string) ([]string, error) {
	handle, ok := types.ParseResourcePath(ref)
	if !ok {
		return nil, fmt.Errorf("invalid image reference")
	}
	var ids []string
	err := db.WithContext(ctx).Table("chunk_images AS i").Group("c.id").Order("CASE WHEN c.chunk_type = 'image_vector' THEN 0 ELSE 1 END").Order("c.id").
		Joins("JOIN chunks c ON c.id = i.chunk_id").
		Joins("JOIN resource_bindings b ON b.owner_id = c.knowledge_id AND b.owner_type = ? AND b.tenant_id = c.tenant_id", types.ResourceOwnerKnowledge).
		Joins("JOIN resources r ON r.id = b.resource_id AND r.tenant_id = c.tenant_id").
		Where("r.handle = ? AND r.state = ? AND r.deleted_at IS NULL", handle, types.ResourceStateActive).
		Where("c.knowledge_base_id IN ?", targets.GetAllKnowledgeBaseIDs()).
		Where("c.deleted_at IS NULL AND c.is_enabled = ? AND i.is_enabled = ?", true, true).
		Where("i.url = ? OR i.original_url = ?", ref, ref).Pluck("c.id", &ids).Error
	return ids, err
}
