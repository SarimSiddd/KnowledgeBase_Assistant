package config

type Server struct {
	Port string
	Host string
}

type Database struct {
	Port     string
	Host     string
	User     string
	Password string
	DBName   string
}

type LangChain struct {
	APIKey  string
	BaseURL string
}

// func LoadConfig() (*Config, error) {
// 	return nil, nil
// }
