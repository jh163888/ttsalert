package sip

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Server          string        `mapstructure:"server"`
	Port            int           `mapstructure:"port"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Domain          string        `mapstructure:"domain"`
	FromUser        string        `mapstructure:"from_user"`
	MaxCallDuration time.Duration `mapstructure:"max_call_duration"`
	RingTimeout     time.Duration `mapstructure:"ring_timeout"`
	MaxRetries      int           `mapstructure:"max_retries"`
	RetryDelay      time.Duration `mapstructure:"retry_delay"`
	LocalPort       int           `mapstructure:"local_port"`
}

type CallRequest struct {
	PhoneNumber string
	AudioFile   string
	AlertID     string
	CallbackURL string
}

type CallResult struct {
	AlertID     string
	PhoneNumber string
	Success     bool
	Error       error
	Duration    time.Duration
	Attempt     int
}

type SIPClient struct {
	config *Config
	logger *logrus.Logger
	mu     sync.Mutex
}

func NewSIPClient(config *Config, logger *logrus.Logger) (*SIPClient, error) {
	return &SIPClient{
		config: config,
		logger: logger,
	}, nil
}

func (c *SIPClient) MakeCall(req CallRequest) CallResult {
	result := CallResult{
		AlertID:     req.AlertID,
		PhoneNumber: req.PhoneNumber,
		Attempt:     1,
	}

	for attempt := 1; attempt <= c.config.MaxRetries; attempt++ {
		result.Attempt = attempt

		success, duration, err := c.attemptCall(req)
		result.Success = success
		result.Duration = duration
		result.Error = err

		if success {
			c.logger.Infof("Call successful to %s, duration: %v", req.PhoneNumber, duration)
			return result
		}

		c.logger.Warnf("Call attempt %d failed to %s: %v", attempt, req.PhoneNumber, err)

		if attempt < c.config.MaxRetries {
			time.Sleep(c.config.RetryDelay)
		}
	}

	return result
}

func (c *SIPClient) attemptCall(req CallRequest) (bool, time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	startTime := time.Now()

	localAddr := fmt.Sprintf("0.0.0.0:%d", c.config.LocalPort)
	if c.config.LocalPort == 0 {
		localAddr = "0.0.0.0:0"
	}

	conn, err := net.Dial("udp", localAddr)
	if err != nil {
		return false, 0, fmt.Errorf("failed to create connection: %w", err)
	}
	defer conn.Close()

	invite := c.buildInvite(req.PhoneNumber)

	if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return false, 0, err
	}

	if _, err := conn.Write([]byte(invite)); err != nil {
		return false, 0, fmt.Errorf("failed to send INVITE: %w", err)
	}

	buffer := make([]byte, 4096)
	if err := conn.SetReadDeadline(time.Now().Add(c.config.RingTimeout)); err != nil {
		return false, 0, err
	}

	n, err := conn.Read(buffer)
	if err != nil {
		return false, 0, fmt.Errorf("failed to receive response: %w", err)
	}

	response := string(buffer[:n])
	if len(response) < 3 {
		return false, 0, fmt.Errorf("invalid SIP response")
	}

	statusCode := response[:3]
	if statusCode != "200" {
		return false, 0, fmt.Errorf("SIP error: %s", response[:50])
	}

	ack := c.buildACK(req.PhoneNumber)
	if _, err := conn.Write([]byte(ack)); err != nil {
		c.logger.Warnf("Failed to send ACK: %v", err)
	}

	c.logger.Infof("Call established with %s", req.PhoneNumber)

	time.Sleep(c.config.MaxCallDuration)

	bye := c.buildBYE(req.PhoneNumber)
	if _, err := conn.Write([]byte(bye)); err != nil {
		c.logger.Warnf("Failed to send BYE: %v", err)
	}

	duration := time.Since(startTime)
	return true, duration, nil
}

func (c *SIPClient) buildInvite(phoneNumber string) string {
	callID := fmt.Sprintf("%d@%s", time.Now().UnixNano(), c.config.Domain)
	cseq := time.Now().UnixNano()
	fromTag := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)

	return fmt.Sprintf(`INVITE sip:%s@%s SIP/2.0
Via: SIP/2.0/UDP %s:%d;branch=z9hG4bK-%d
Max-Forwards: 70
From: <sip:%s@%s>;tag=%s
To: <sip:%s@%s>
Call-ID: %s
CSeq: %d INVITE
Contact: <sip:%s@%s:%d>
Content-Type: application/sdp
Content-Length: 0

`, phoneNumber, c.config.Domain, c.config.Server, c.config.Port, cseq,
		c.config.FromUser, c.config.Domain, fromTag,
		phoneNumber, c.config.Domain, callID,
		cseq, c.config.FromUser, c.config.Domain, c.config.LocalPort)
}

func (c *SIPClient) buildACK(phoneNumber string) string {
	return fmt.Sprintf("ACK sip:%s@%s SIP/2.0\r\n\r\n", phoneNumber, c.config.Domain)
}

func (c *SIPClient) buildBYE(phoneNumber string) string {
	return fmt.Sprintf("BYE sip:%s@%s SIP/2.0\r\n\r\n", phoneNumber, c.config.Domain)
}

func (c *SIPClient) HealthCheck() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Debug("SIP health check")
	return true
}
