package logwatcher

import (
	"context"
	"github.com/nxadm/tail"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/config"
	"github.com/otterize/network-mapper/src/kafka-watcher/pkg/mapperclient"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"sync"
	"time"
)

type LogFileWatcher struct {
	baseWatcher

	authzFilePath string
	server        types.NamespacedName
}

func NewLogFileWatcher(mapperClient mapperclient.MapperClient, authzFilePath string, server types.NamespacedName) (*LogFileWatcher, error) {
	w := &LogFileWatcher{
		baseWatcher: baseWatcher{
			mu:           sync.Mutex{},
			seen:         SeenRecordsStore{},
			mapperClient: mapperClient,
		},
		authzFilePath: authzFilePath,
		server:        server,
	}

	return w, nil
}

func (w *LogFileWatcher) RunForever(ctx context.Context) {
	go w.watchForever(ctx)

	for {
		time.Sleep(viper.GetDuration(config.KafkaCooldownIntervalKey))

		if err := w.reportResults(ctx); err != nil {
			logrus.WithError(err).Errorf("Failed reporting watcher results to mapper")
		}
	}
}

func (w *LogFileWatcher) watchForever(ctx context.Context) {
	t, err := tail.TailFile(w.authzFilePath, tail.Config{Follow: true, ReOpen: true, MustExist: false, Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd}})

	if err != nil {
		logrus.WithError(err).Panic()
	}

	for line := range t.Lines {
		w.processLogRecord(w.server, line.Text)
	}
}
