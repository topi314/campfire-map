package campfire

import "fmt"

const DefaultRealityChannelID = "da83476a-c4da-4312-a610-a4f2fc2c37f0"

type Config struct {
	URL              string `toml:"url"`
	MaxRetries       int    `toml:"max_retries"`
	RealityChannelID string `toml:"reality_channel_id"`
	S2CellLevel      int    `toml:"s2_cell_level"`
}

func (c Config) String() string {
	return fmt.Sprintf("\n URL: %s\n MaxRetries: %d\n RealityChannelID: %s\n S2CellLevel: %d",
		c.URL,
		c.MaxRetries,
		c.RealityChannelID,
		c.S2CellLevel,
	)
}
