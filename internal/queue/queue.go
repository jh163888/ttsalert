package queue

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/ttsalert/ttsalert/internal/handler"
	"github.com/ttsalert/ttsalert/internal/sip"
	"github.com/ttsalert/ttsalert/internal/tts"
)

type AlertQueue struct {
	queue     chan handler.Alert
	ttsSvc    *tts.EdgeTTSService
	sipClient *sip.SIPClient
	logger    *logrus.Logger
	wg        sync.WaitGroup
}

func NewAlertQueue(size int, ttsSvc *tts.EdgeTTSService, sipClient *sip.SIPClient, logger *logrus.Logger) *AlertQueue {
	return &AlertQueue{
		queue:     make(chan handler.Alert, size),
		ttsSvc:    ttsSvc,
		sipClient: sipClient,
		logger:    logger,
	}
}

func (q *AlertQueue) Enqueue(alert handler.Alert) error {
	select {
	case q.queue <- alert:
		q.logger.Debugf("Alert queued: %s", alert.ID)
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

func (q *AlertQueue) Start(workers int) {
	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
	q.logger.Infof("Started %d queue workers", workers)
}

func (q *AlertQueue) worker(id int) {
	defer q.wg.Done()
	q.logger.Debugf("Worker %d started", id)

	for alert := range q.queue {
		q.processAlert(alert)
	}
}

func (q *AlertQueue) processAlert(alert handler.Alert) {
	q.logger.Infof("Processing alert: %s - %s", alert.ID, alert.Title)

	text := q.buildAlertText(alert)

	audioFile, err := q.ttsSvc.GenerateSpeech(text, alert.ID)
	if err != nil {
		q.logger.Errorf("Failed to generate speech for alert %s: %v", alert.ID, err)
		return
	}

	for _, phone := range alert.PhoneNumbers {
		req := sip.CallRequest{
			PhoneNumber: phone,
			AudioFile:   audioFile,
			AlertID:     alert.ID,
		}

		result := q.sipClient.MakeCall(req)

		if result.Success {
			q.logger.Infof("Call completed to %s, duration: %v", phone, result.Duration)
		} else {
			q.logger.Errorf("Call failed to %s: %v", phone, result.Error)
		}
	}
}

func (q *AlertQueue) buildAlertText(alert handler.Alert) string {
	source := ""
	switch alert.Source {
	case "zabbix":
		source = "Zabbix 告警"
	case "opm":
		source = "卓豪 OPM 告警"
	default:
		source = "系统告警"
	}

	text := fmt.Sprintf("%s，%s，主机：%s，%s", source, alert.Title, alert.Host, alert.Message)

	if alert.Severity != "" {
		text = fmt.Sprintf("告警级别%s，%s", alert.Severity, text)
	}

	return text
}

func (q *AlertQueue) Stop() {
	close(q.queue)
	q.wg.Wait()
	q.logger.Info("Queue workers stopped")
}

func (q *AlertQueue) Size() int {
	return len(q.queue)
}
