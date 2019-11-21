package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/micro/cli"
	"github.com/micro/go-micro/config/cmd"
	gorun "github.com/micro/go-micro/runtime"
	"github.com/micro/go-micro/util/log"
	"github.com/micro/micro/internal/update"
)

type initNotifier struct {
	gorun.Notifier
	services []string
}

func (i *initNotifier) Notify() (<-chan gorun.Event, error) {
	ch, err := i.Notifier.Notify()
	if err != nil {
		return nil, err
	}

	// create new event channel
	evChan := make(chan gorun.Event, 32)

	go func() {
		for ev := range ch {
			// fire an event per service
			for _, service := range i.services {
				evChan <- gorun.Event{
					Service:   service,
					Version:   ev.Version,
					Timestamp: ev.Timestamp,
					Type:      ev.Type,
				}
			}
		}

		// we've reached the end
		close(evChan)
	}()

	return evChan, nil
}

func initNotify(n gorun.Notifier, services []string) gorun.Notifier {
	return &initNotifier{n, services}
}

func initCommand(context *cli.Context) {
	log.Name("init")

	if len(context.Args()) > 0 {
		cli.ShowSubcommandHelp(context)
		os.Exit(1)
	}

	services := []string{
		"api", // :8080
	}

	// create new micro runtime
	muRuntime := cmd.DefaultCmd.Options().Runtime

	// Use default update notifier
	notifier := update.NewNotifier(BuildDate)
	wrapped := initNotify(notifier, services)

	// specify with a notifier that fires
	// individual events for each service
	options := []gorun.Option{
		gorun.WithNotifier(wrapped),
	}
	(*muRuntime).Init(options...)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	log.Info("Starting service runtime")

	// start the runtime
	if err := (*muRuntime).Start(); err != nil {
		log.Fatal(err)
	}

	log.Info("Service runtime started")

	select {
	case <-shutdown:
		log.Info("Shutdown signal received")
		log.Info("Stopping service runtime")
	}

	// stop all the things
	if err := (*muRuntime).Stop(); err != nil {
		log.Fatal(err)
	}

	log.Info("Service runtime shutdown")

	// exit success
	os.Exit(0)
}
