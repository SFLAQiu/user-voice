package config

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
	Admin      AdminSeedConfig  `mapstructure:"admin"`
	LoginLimit LoginLimitConfig `mapstructure:"login_limit"`
	Log        LogConfig        `mapstructure:"log"`
	CORS       CORSConfig       `mapstructure:"cors"`
}

type ServerConfig struct {
	Addr         string        `mapstructure:"addr"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type JWTConfig struct {
	Secret string        `mapstructure:"secret"`
	Expire time.Duration `mapstructure:"expire"`
}

type EncryptionConfig struct {
	Key string `mapstructure:"key"` // base64 32 bytes
}

type AdminSeedConfig struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type LoginLimitConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	Window      time.Duration `mapstructure:"window"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// Validate 校验向导提交的关键配置项合法性。
func Validate(jwtSecret, encryptionKey string) error {
	if len(jwtSecret) < 16 {
		return fmt.Errorf("jwt secret 至少 16 字符")
	}
	key, err := base64.StdEncoding.DecodeString(encryptionKey)
	if err != nil {
		return fmt.Errorf("加密密钥不是合法 base64: %w", err)
	}
	if len(key) != 32 {
		return fmt.Errorf("加密密钥需为 32 字节的 base64，当前 %d 字节", len(key))
	}
	return nil
}

// Load reads config from path; env vars override using FEEDBACK_<SECTION>_<KEY>.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("FEEDBACK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", time.Hour)
	v.SetDefault("jwt.expire", 8*time.Hour)
	v.SetDefault("login_limit.max_attempts", 5)
	v.SetDefault("login_limit.window", 5*time.Minute)
	v.SetDefault("log.level", "info")
}

func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("database.dsn is required")
	}
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("jwt.secret must be at least 16 chars")
	}
	if c.Encryption.Key == "" {
		return fmt.Errorf("encryption.key is required (base64 32 bytes)")
	}
	return nil
}
