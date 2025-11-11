package main

import (
	"flag"
	"strings"

	"github.com/rs/zerolog/log"
	"k8s.io/klog/v2"
)

func initKlogBridge(logKubernetesHttpRequests bool) {
	klog.InitFlags(nil)
	_ = flag.Set("logtostderr", "false")
	if logKubernetesHttpRequests {
		_ = flag.Set("vmodule", "round_trippers=6")
	}
	flag.Parse()
	klog.SetOutput(zeroLogWriter{})
}

type zeroLogWriter struct {
	includes []string
}

func (e zeroLogWriter) Write(p []byte) (int, error) {
	logMessage := string(p)
	log.Info().Msgf("klog --> %s", strings.Replace(logMessage, "\n", "", -1))
	return len(p), nil
}
