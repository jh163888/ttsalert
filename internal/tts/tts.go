package tts

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Voice       string `mapstructure:"voice"`
	Rate        string `mapstructure:"rate"`
	Volume      string `mapstructure:"volume"`
	Pitch       string `mapstructure:"pitch"`
	OutputDir   string `mapstructure:"output_dir"`
	AudioFormat string `mapstructure:"audio_format"`
	UseEdgeTTS  bool   `mapstructure:"use_edgetts"`
}

type EdgeTTSService struct {
	config *Config
	logger *logrus.Logger
}

func NewEdgeTTSService(config *Config, logger *logrus.Logger) (*EdgeTTSService, error) {
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &EdgeTTSService{
		config: config,
		logger: logger,
	}, nil
}

func (s *EdgeTTSService) GenerateSpeech(text, alertID string) (string, error) {
	outputFile := filepath.Join(s.config.OutputDir, fmt.Sprintf("%s.%s", alertID, s.config.AudioFormat))

	if _, err := os.Stat(outputFile); err == nil {
		s.logger.Infof("Audio file already exists: %s", outputFile)
		return outputFile, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if s.config.UseEdgeTTS {
		cmd = exec.CommandContext(ctx, "edge-tts",
			"--voice", s.config.Voice,
			"--text", text,
			"--rate", s.config.Rate,
			"--volume", s.config.Volume,
			"--pitch", s.config.Pitch,
			"--write-media", outputFile,
		)
	} else {
		cmd = exec.CommandContext(ctx, "espeak",
			"-v", "zh",
			"-s", "150",
			"-w", outputFile,
			text,
		)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate speech: %w, stderr: %s", err, stderr.String())
	}

	s.logger.Infof("Generated speech: %s", outputFile)
	return outputFile, nil
}

func (s *EdgeTTSService) Cleanup(oldHours int) error {
	cutoff := time.Now().Add(-time.Duration(oldHours) * time.Hour)

	entries, err := os.ReadDir(s.config.OutputDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(s.config.OutputDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				s.logger.Warnf("Failed to remove old file %s: %v", filePath, err)
			} else {
				s.logger.Debugf("Cleaned up old file: %s", filePath)
			}
		}
	}

	return nil
}
