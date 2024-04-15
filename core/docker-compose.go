package core

import (
	"context"
	"time"

	"github.com/rotisserie/eris"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

type DockerCompose struct {
	compose.ComposeStack
	exporterService string
	startupTimeout  time.Duration
}

func NewDockerCompose(composeFilePath string, service string, startupTimeout time.Duration) (*DockerCompose, error) {
	compose, err := compose.NewDockerCompose(composeFilePath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create docker compose")
	}

	return &DockerCompose{
		ComposeStack:    compose,
		exporterService: service,
		startupTimeout:  startupTimeout,
	}, nil

}

// Setup starts the docker-compose stack.
func (c *DockerCompose) Setup(ctx context.Context) error {
	err := c.Up(
		ctx, compose.Wait(true), compose.RemoveOrphans(true),
	)

	if err != nil {
		return eris.Wrap(err, "failed to wait for service")
	}

	return nil
}

// TearDown stops and removes the docker-compose stack.
func (c *DockerCompose) TearDown(ctx context.Context) error {
	err := c.Down(ctx, compose.RemoveOrphans(true))
	if err != nil {
		return eris.Wrap(err, "failed to tear down")
	}

	return nil
}

// Start wait for the service to be ready and returns the endpoint.
func (c *DockerCompose) Start(ctx context.Context) (string, error) {
	container, err := c.ServiceContainer(ctx, c.exporterService)
	if err != nil {
		return "", eris.Wrap(err, "failed to get service container")
	}

	strategy := wait.ForExposedPort().WithStartupTimeout(c.startupTimeout)
	err = strategy.WaitUntilReady(ctx, container)
	if err != nil {
		return "", eris.Wrap(err, "failed to wait for service")
	}

	endpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		return "", eris.Wrap(err, "failed to get endpoint")
	}

	return endpoint, nil
}
