package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jh163888/ttsalert/internal/handler"
	"github.com/jh163888/ttsalert/internal/queue"
	"github.com/jh163888/ttsalert/internal/sip"
	"github.com/jh163888/ttsalert/internal/tts"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	TTS     tts.Config    `mapstructure:"tts"`
	SIP     sip.Config    `mapstructure:"sip"`
	Queue   QueueConfig   `mapstructure:"queue"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type QueueConfig struct {
	Size    int `mapstructure:"size"`
	Workers int `mapstructure:"workers"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

var version = "dev"

func main() {
	if err := initConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(config.Logging)
	logger.Infof("TTS Alert starting (version: %s)", version)

	ttsSvc, err := tts.NewEdgeTTSService(&config.TTS, logger)
	if err != nil {
		logger.Fatalf("Failed to create TTS service: %v", err)
	}

	sipClient, err := sip.NewSIPClient(&config.SIP, logger)
	if err != nil {
		logger.Fatalf("Failed to create SIP client: %v", err)
	}

	alertQueue := queue.NewAlertQueue(
		config.Queue.Size,
		ttsSvc,
		sipClient,
		logger,
	)

	alertQueue.Start(config.Queue.Workers)

	router := mux.NewRouter()
	webhookHandler := handler.NewWebhookHandler(alertQueue, logger)
	webhookHandler.RegisterRoutes(router)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Infof("Starting server on %s:%d", config.Server.Host, config.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	logger.Info("TTS Alert is ready")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	alertQueue.Stop()

	logger.Info("Server exited")
}

func initConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/ttsalert")
	viper.AddConfigPath("$HOME/.ttsalert")
	viper.AddConfigPath(".")

	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("queue.size", 100)
	viper.SetDefault("queue.workers", 3)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("tts.voice", "zh-CN-XiaoxiaoNeural")
	viper.SetDefault("tts.rate", "+0%")
	viper.SetDefault("tts.volume", "+0%")
	viper.SetDefault("tts.pitch", "+0Hz")
	viper.SetDefault("tts.output_dir", "/var/lib/ttsalert/audio")
	viper.SetDefault("tts.audio_format", "mp3")
	viper.SetDefault("tts.use_edgetts", true)
	viper.SetDefault("sip.port", 5060)
	viper.SetDefault("sip.local_port", 5060)
	viper.SetDefault("sip.max_call_duration", 120*time.Second)
	viper.SetDefault("sip.ring_timeout", 30*time.Second)
	viper.SetDefault("sip.max_retries", 3)
	viper.SetDefault("sip.retry_delay", 5*time.Second)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		fmt.Fprintf(os.Stderr, "Loaded config from: %s\n", configFile)
	}

	return nil
}

func loadConfig() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func setupLogger(config LoggingConfig) *logrus.Logger {
	logger := logrus.New()

	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	return logger
}
