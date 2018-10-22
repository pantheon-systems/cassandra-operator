package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"

	opsdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	stub "github.com/pantheon-systems/cassandra-operator/pkg/stub"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/pantheon-systems/cassandra-operator/version"
	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	allNamespaces = ""
	resource      = "database.pantheon.io/v1alpha1"
	kind          = "CassandraCluster"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		select {
		case <-signals:
			cancel()
		case <-ctx.Done():
			os.Exit(0)
		}
	}()

	resyncPeriod := flag.Duration("resync", 20*time.Second, "Resync period")
	debug := flag.Bool("debug", false, "debug level logging")
	versionTaint := flag.String("version-taint", "", "sets and enables a version taint to run a private controller")
	flag.Parse()

	if versionTaint != nil && *versionTaint != "" {
		version.Version = fmt.Sprintf("%s-%s", version.Version, *versionTaint)
	}

	printVersion()

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.999Z07:00", // RFC3339 at millisecond precision
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime: "@timestamp",
			logrus.FieldKeyMsg:  "message",
		},
	})

	if *debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Logging level set to DEBUG")
	}

	kubeClient := k8s.NewOperatorSdkClient()
	nodetoolClient := nodetool.NewExecutor(kubeClient)
	handler := stub.NewHandler(kubeClient, nodetoolClient)

	// Register primary watcher and handler for CassandraCluster CRD
	logrus.Infof("Watching %s, %s, all namespaces, %d", resource, kind, *resyncPeriod)
	opsdk.Watch(resource, kind, allNamespaces, *resyncPeriod)
	opsdk.Watch("v1", "Pod", allNamespaces, 0, opsdk.WithLabelSelector("type=cassandra-node"))
	opsdk.Handle(handler)
	opsdk.Run(ctx)
}

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
	logrus.Infof("cassandra-operator Version: %v", version.Version)
}
