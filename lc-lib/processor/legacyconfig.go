package processor

import (
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/processor/ast"
)

// LegacyConfig contains legacy configuration for a processor pipeline
// Users should use `pipeline` instead of `pipelines` which is Config
type LegacyConfig struct {
	Pipeline []*LegacyConfigASTEntry `config:",embed_slice" json:",omitempty"`

	ast []ast.ProcessNode
}

// LegacyConfigASTEntry is a configuration entry we need to parse into an ASTEntry
type LegacyConfigASTEntry struct {
	Logic map[string]interface{} `config:",collect_unused" json:",omitempty"`
	path  string
}

// FetchLegacyConfig returns the processor configuration from a Config structure
func FetchLegacyConfig(cfg *config.Config) *LegacyConfig {
	return cfg.Section("pipelines").(*LegacyConfig)
}
