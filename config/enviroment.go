package config

// Env string
type Env string

// Enum of Environment
const (
	Local Env = "local"
	Test  Env = "test"
	Dev   Env = "dev"
	Prod  Env = "prod"
)

// GetFile for Env
func (e Env) GetFile() string {
	return string(e) + ".json"
}
