// config/config.go

package config

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	LangChain LangChainConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Port     string
	Host     string
	User     string
	Password string
	DBName   string
}

type LangChainConfig struct {
	APIKey  string
	BaseURL string
}

func LoadConfig() (*Config, error) {

}
