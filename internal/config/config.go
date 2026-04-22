package config

import "os"

type Config struct {
	Addr         string
	DataDir      string
	OpenAIAPIKey string
	OpenAIModel  string
	SessionDays  int
}

func Load() Config {
	addr := os.Getenv("APP_ADDR")
	if addr == "" { addr = ":8090" }
	dataDir := os.Getenv("APP_DATA_DIR")
	if dataDir == "" { dataDir = "data" }
	model := os.Getenv("OPENAI_MODEL")
	if model == "" { model = "gpt-4o-mini" }
	return Config{
		Addr: addr,
		DataDir: dataDir,
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		OpenAIModel: model,
		SessionDays: 30,
	}
}
