package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/pantheon-systems/cassandra-operator/pkg/apis"
	"github.com/pantheon-systems/cassandra-operator/pkg/controller"
	"github.com/prometheus/common/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
	logrus.Infof("cassandra-operator Version: %v", version.Version)
}

func main() {
	//resyncPeriod := flag.Duration("resync", 20*time.Second, "Resync period")
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

	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		log.Fatalf("failed to get watch namespace: %v", err)
	}

	// TODO: Expose metrics port after SDK uses controller-runtime's dynamic client
	// sdk.ExposeMetricsPort()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	ctx, _ := setupSignalHandler()

	// Setup all Controllers
	if err := controller.AddToManager(ctx, mgr); err != nil {
		log.Fatal(err)
	}

	log.Print("Starting the Cmd.")

	// Start the Cmd
	log.Fatal(mgr.Start(ctx.Done()))
}

func setupSignalHandler() (context.Context, context.CancelFunc) {
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
	return ctx, cancel
}
