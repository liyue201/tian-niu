package shared

type ModelConfig struct {
	BaseURL       string `yaml:"base_url"`
	ApiKey        string `yaml:"api_key"`
	Model         string `yaml:"model"`
	ContextWindow int    `yaml:"context_window"`
}

type VectorDBConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Database  string `yaml:"database"`
	Dimension int    `yaml:"dimension"`
}

type EmbeddingServiceConfig struct {
	APIKey     string `yaml:"api_key"`
	BaseURL    string `yaml:"base_url"`
	Model      string `yaml:"model"`
	Dimensions int    `yaml:"dimensions"`
}

type RerankServiceConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type MemoryStrategyConfig struct {
	QuickSaveRounds          int     `yaml:"quick_save_rounds"`
	RegularSaveRounds        int     `yaml:"regular_save_rounds"`
	ForceSaveRounds          int     `yaml:"force_save_rounds"`
	MinTokenThreshold        int     `yaml:"min_token_threshold"`
	TopicSimilarityThreshold float32 `yaml:"topic_similarity_threshold"`
}

type LongTermMemoryConfig struct {
	Enabled          bool                   `yaml:"enabled"`
	VectorDB         VectorDBConfig         `yaml:"vector_db"`
	EmbeddingService EmbeddingServiceConfig `yaml:"embedding_service"`
	RerankService    RerankServiceConfig    `yaml:"rerank_service"`
	Strategy         MemoryStrategyConfig   `yaml:"strategy"`
}
