package logrus2loki

import (
	"fmt"
	"strings"
	"time"

	"github.com/afiskon/promtail-client/promtail"
	"github.com/sirupsen/logrus"
)

var supportedLevels = []logrus.Level{logrus.DebugLevel, logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel}

// Config defines configuration for hook for Loki
type Config struct {
	URL                string
	Labels             map[string]string
	LevelName          string
	BatchWait          time.Duration
	BatchEntriesNumber int
}

func (c *Config) setDefault() {
	if c.URL == "" {
		c.URL = "http://localhost:3100/api/prom/push"
	}
	if c.Labels == nil {
		c.Labels = map[string]string{
			"source": "test",
			"job":    "job",
		}
	}
	if c.BatchWait == time.Second {
		c.BatchWait = 5 * time.Second
	}
	if c.BatchEntriesNumber == 0 {
		c.BatchEntriesNumber = 10000
	}

}

// genLabelsWithLogLevel generate available labels of loki from level and the label dict you defined
func (c *Config) genLabelsWithLogLevel(level string) string {
	c.Labels[c.LevelName] = level
	labelsList := []string{}
	for k, v := range c.Labels {
		labelsList = append(labelsList, fmt.Sprintf(`%s="%s"`, k, v))
	}
	labelString := fmt.Sprintf(`{%s}`, strings.Join(labelsList, ", "))
	return labelString
}

type Hook struct {
	clients map[logrus.Level]promtail.Client
}

// NewHook creates a new hook for Loki
func NewHook(c *Config) (*Hook, error) {
	var err error
	if c == nil {
		c = &Config{}
	}
	c.setDefault()
	conf := promtail.ClientConfig{
		PushURL:            c.URL,
		BatchWait:          c.BatchWait,
		BatchEntriesNumber: c.BatchEntriesNumber,
		SendLevel:          promtail.INFO,
		PrintLevel:         promtail.ERROR,
	}

	// create different promtail client instance
	clients := make(map[logrus.Level]promtail.Client)
	for _, v := range supportedLevels {
		conf.Labels = c.genLabelsWithLogLevel(v.String())
		clients[v], err = promtail.NewClientJson(conf)
		if err != nil {
			return nil, fmt.Errorf("unable to init promtail client: %v", err)
		}
	}
	return &Hook{
		clients: clients,
	}, nil
}

// Fire implements interface for logrus
func (hook *Hook) Fire(entry *logrus.Entry) error {
	msg, err := entry.String()
	if err != nil {
		return err
	}
	switch entry.Level {
	case logrus.DebugLevel:
		hook.clients[entry.Level].Debugf(msg)
	case logrus.InfoLevel:
		hook.clients[entry.Level].Infof(msg)
	case logrus.WarnLevel:
		hook.clients[entry.Level].Warnf(msg)
	case logrus.ErrorLevel:
		hook.clients[entry.Level].Errorf(msg)
	default:
		return fmt.Errorf("unknown log level")
	}
	return nil
}

// Levels retruns supported levels
func (hook *Hook) Levels() []logrus.Level {
	return supportedLevels
}
