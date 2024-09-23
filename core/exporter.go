package core

import (
	"context"
	"fmt"
	"time"

	"github.com/rotisserie/eris"
)

type DockerComposeExporter struct {
	dockerCompose   *DockerCompose
	exporterService string
	exporterHost    string
	exporterPort    string
	startupTimeout  time.Duration
}

// Start wait for the service to be ready and returns the endpoint.
func (e *DockerComposeExporter) Start(ctx context.Context) (string, error) {
	_, err := e.dockerCompose.ServiceContainer(ctx, e.exporterService)
	if err != nil {
		return "", eris.Wrap(err, "failed to get service container")
	}

	//strategy := wait.ForHealthCheck().WithStartupTimeout(e.startupTimeout)
	//err = strategy.WaitUntilReady(ctx, container)
	//if err != nil {
	//	return "", eris.Wrapf(err, "failed to wait for service container: %s", e.exporterService)
	//}

	//endpoint, err := container.Endpoint(ctx, "http")
	endpoint := fmt.Sprintf("http://%s:%s", e.exporterHost, e.exporterPort)
	if err != nil {
		return "", eris.Wrap(err, "failed to get endpoint")
	}

	return endpoint, nil
}

func NewDockerComposeExporter(dockerCompose *DockerCompose, exporterService string, exporterHost string, exporterPort string, startupTimeout time.Duration) *DockerComposeExporter {
	return &DockerComposeExporter{
		dockerCompose:   dockerCompose,
		exporterService: exporterService,
		exporterHost:    exporterHost,
		exporterPort:    exporterPort,
		startupTimeout:  startupTimeout,
	}
}

type ExternalExporter struct {
	baseurl string
}

func (e *ExternalExporter) Start(ctx context.Context) (string, error) {
	return e.baseurl, nil
}

func NewExternalExporter(baseurl string) *ExternalExporter {
	return &ExternalExporter{
		baseurl: baseurl,
	}
}
