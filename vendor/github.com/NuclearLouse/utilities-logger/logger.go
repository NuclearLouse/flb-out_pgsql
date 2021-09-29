package logger

import (
	"io"
	"os"

	env "github.com/NuclearLouse/utilities-environ"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/orandin/lumberjackrus"
	"github.com/sirupsen/logrus"
)

// Config ...
type Config struct {
	Level         string
	LogFile       string `ini:"filepath"`
	ErrFile       string `ini:"error_file"`
	MaxSize       int
	MaxBackup     int
	MaxAge        int
	Compress      bool
	Localtime     bool
	FormatTime    string
	ShowFullLevel bool `ini:"show_full_lvl"`
}

// New ...функция возвращающая указатель на структуру Logger. Если не передан укзатель на Config,
// настройки будут читаться из переменных окружения.
func New(config ...*Config) (*logrus.Logger, error) {

	var cfg *Config

	if config == nil {
		cfg = envConfig()
	} else {
		cfg = config[0]
	}

	log := logrus.New()

	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.TraceLevel
	}

	log.SetLevel(level)

	format := &nested.Formatter{
		TimestampFormat: cfg.FormatTime,
		ShowFullLevel:   cfg.ShowFullLevel,
		// NoColors:        true,
	}
	log.SetFormatter(format)

	if cfg.LogFile != "" {
		format.NoColors = true
		log.SetFormatter(format)
		if cfg.ErrFile == "" {
			file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
			if err != nil {
				return log, err
			}
			log.SetOutput(io.MultiWriter(file, os.Stdout))

		} else {
			hook, err := lumberjackrus.NewHook(
				hookFile(cfg, cfg.LogFile),
				logrus.DebugLevel,
				format,
				&lumberjackrus.LogFileOpts{
					logrus.WarnLevel:  hookFile(cfg, cfg.ErrFile),
					logrus.ErrorLevel: hookFile(cfg, cfg.ErrFile),
					logrus.FatalLevel: hookFile(cfg, cfg.ErrFile),
				},
			)

			if err != nil {
				return nil, err
			}

			log.AddHook(hook)
		}
	}

	return log, nil
}

func hookFile(cfg *Config, file string) *lumberjackrus.LogFile {
	return &lumberjackrus.LogFile{
		Filename:   file,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackup,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  cfg.Localtime,
	}
}

func envConfig() *Config {

	return &Config{
		Level:         env.GetEnv("LOG_LVL", "trace"),
		LogFile:       env.GetEnv("LOG_FILE", ""),
		ErrFile:       env.GetEnv("LOG_ERR_FILE", ""),
		MaxSize:       env.GetEnvAsInt("LOG_MAX_SIZE", 1),
		MaxBackup:     env.GetEnvAsInt("LOG_MAX_BACKUP", 3),
		MaxAge:        env.GetEnvAsInt("LOG_MAX_AGE", 1),
		Compress:      env.GetEnvAsBool("LOG_COMPRESS", true),
		Localtime:     env.GetEnvAsBool("LOG_LOCALTIME", true),
		FormatTime:    env.GetEnv("LOG_FORMAT_TIME", "2006-01-02 15:04:05.000"),
		ShowFullLevel: env.GetEnvAsBool("LOG_FULL_LVL", false),
	}
}

// DefaultConfig ...
func DefaultConfig() *Config {
	return &Config{
		Level:      "trace",
		MaxSize:    1,
		MaxBackup:  3,
		MaxAge:     1,
		Compress:   true,
		Localtime:  true,
		FormatTime: "2006-01-02 15:04:05.000",
	}
}
