package service

import (
	"context"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestKnowledgeImageRequiresDocumentResourceOwnership(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{
		"CREATE TABLE chunks (id TEXT, knowledge_id TEXT, knowledge_base_id TEXT, tenant_id INTEGER, deleted_at DATETIME, is_enabled BOOLEAN, chunk_type TEXT)",
		"CREATE TABLE chunk_images (chunk_id TEXT,url TEXT, original_url TEXT,is_enabled BOOLEAN)",
		"CREATE TABLE resource_bindings (resource_id TEXT, owner_id TEXT, owner_type TEXT, tenant_id INTEGER)",
		"CREATE TABLE resources (id TEXT, handle TEXT, tenant_id INTEGER, state TEXT, deleted_at DATETIME)",
		"INSERT INTO chunks VALUES ('c','doc','kb',7,NULL,1,'image_caption')",
		"INSERT INTO chunk_images VALUES ('c','resource://abcdefghijklmnopqrstuv','',1)",
		"INSERT INTO resources VALUES ('r','abcdefghijklmnopqrstuv',7,'active',NULL)",
		"INSERT INTO resource_bindings VALUES ('r','other-doc','knowledge',7)",
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	targets := types.SearchTargets{{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: "kb", TenantID: 7}}
	lookup := func() []string {
		ids, err := knowledgeImageChunkIDs(context.Background(), db, targets, "resource://abcdefghijklmnopqrstuv")
		require.NoError(t, err)
		return ids
	}
	require.Empty(t, lookup(), "a forged image URL must not authorize a resource owned by another document")
	require.NoError(t, db.Exec("UPDATE resource_bindings SET owner_id='doc'").Error)
	require.Equal(t, []string{"c"}, lookup())
	require.NoError(t, db.Exec("UPDATE resources SET state='pending_delete'").Error)
	require.Empty(t, lookup())
	require.NoError(t, db.Exec("UPDATE resources SET state='active',tenant_id=8").Error)
	require.Empty(t, lookup())
}
