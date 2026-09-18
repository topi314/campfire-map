package wayfarer

import "fmt"

const DefaultURL = "https://wayfarer.scopely.com/api/v1/vault/mapview/gcs"

type Config struct {
	URL        string `toml:"url"`
	MaxRetries int    `toml:"max_retries"`
}

func (c Config) String() string {
	url := c.URL
	if url == "" {
		url = DefaultURL
	}
	return fmt.Sprintf("\n URL: %s\n MaxRetries: %d", url, c.MaxRetries)
}
