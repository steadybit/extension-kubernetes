package main

import (
	"flag"
	"github.com/rs/zerolog/log"
	"k8s.io/klog/v2"
	"strings"
)

func initKlogBridge(logKubernetesHttpRequests bool) {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "false")
	if logKubernetesHttpRequests {
		flag.Set("vmodule", "round_trippers=6")
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
