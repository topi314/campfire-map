package server

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/topi314/campfire-export/server/campfire"
	"github.com/topi314/campfire-export/server/wayfarer"
)

func LoadConfig(cfgPath string) (Config, error) {
	file, err := os.Open(cfgPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	cfg := defaultConfig()
	if _, err = toml.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to decode config file: %w", err)
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:     slog.LevelInfo,
			Format:    LogFormatText,
			AddSource: false,
		},
		Server: ServerConfig{
			Addr:        ":8080",
			CORSOrigins: "http://localhost:3000,http://127.0.0.1:3000",
		},
		Campfire: campfire.Config{
			URL:              "https://niantic-social-api.nianticlabs.com/graphql",
			MaxRetries:       3,
			RealityChannelID: campfire.DefaultRealityChannelID,
			S2CellLevel:      15,
		},
		Wayfarer: wayfarer.Config{
			URL:        wayfarer.DefaultURL,
			MaxRetries: 3,
		},
		Cache: CacheConfig{
			TTL: 24 * time.Hour,
		},
		Limits: LimitsConfig{
			MaxCells:      192,
			MaxBBoxSpan:   0.35,
			MaxExportPOIs: 8000,
		},
	}
}

type Config struct {
	Log      LogConfig       `toml:"log"`
	Server   ServerConfig    `toml:"server"`
	Campfire campfire.Config `toml:"campfire"`
	Wayfarer wayfarer.Config `toml:"wayfarer"`
	Cache    CacheConfig     `toml:"cache"`
	Limits   LimitsConfig    `toml:"limits"`
	Basemaps BasemapsConfig  `toml:"basemaps"`
}

func (c Config) String() string {
	return fmt.Sprintf("Log: %s\nServer: %s\nCampfire: %s\nWayfarer: %s\nCache: %s\nLimits: %s\nBasemaps: %s",
		c.Log,
		c.Server,
		c.Campfire,
		c.Wayfarer,
		c.Cache,
		c.Limits,
		c.Basemaps,
	)
}

type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

type LogConfig struct {
	Level     slog.Level `toml:"level"`
	Format    LogFormat  `toml:"format"`
	AddSource bool       `toml:"add_source"`
}

func (c LogConfig) String() string {
	return fmt.Sprintf("\n Level: %s\n Format: %s\n AddSource: %t",
		c.Level,
		c.Format,
		c.AddSource,
	)
}

type ServerConfig struct {
	Addr        string `toml:"addr"`
	CORSOrigins string `toml:"cors_origins"`
}

func (c ServerConfig) String() string {
	return fmt.Sprintf("\n Addr: %s\n CORSOrigins: %s",
		c.Addr,
		c.CORSOrigins,
	)
}

type CacheConfig struct {
	TTL time.Duration `toml:"ttl"`
}

func (c CacheConfig) String() string {
	return fmt.Sprintf("\n TTL: %s", c.TTL)
}

type LimitsConfig struct {
	MaxCells      int     `toml:"max_cells"`
	MaxBBoxSpan   float64 `toml:"max_bbox_span"`
	MaxExportPOIs int     `toml:"max_export_pois"`
}

func (c LimitsConfig) String() string {
	return fmt.Sprintf("\n MaxCells: %d\n MaxBBoxSpan: %g\n MaxExportPOIs: %d",
		c.MaxCells,
		c.MaxBBoxSpan,
		c.MaxExportPOIs,
	)
}

type BasemapsConfig struct {
	CartoAPIKey string `toml:"carto_api_key"`
}

// CartoKey is served to the browser, which appends it to CARTO tile URLs.
func (c BasemapsConfig) CartoKey() string {
	return strings.TrimSpace(c.CartoAPIKey)
}

func (c BasemapsConfig) String() string {
	set := "unset"
	if c.CartoKey() != "" {
		set = "set"
	}
	return fmt.Sprintf("\n CartoAPIKey: %s", set)
}
