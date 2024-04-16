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

	RemoveAllImages bool
}

func (c *DockerCompose) String() string {
	return "DockerCompose"
}

func NewDockerCompose(composeFilePath string, RemoveAllImages bool) (*DockerCompose, error) {
	compose, err := compose.NewDockerCompose(composeFilePath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create docker compose")
	}

	return &DockerCompose{
		ComposeStack:    compose,
		RemoveAllImages: RemoveAllImages,
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
	var removeImages compose.RemoveImages
	if c.RemoveAllImages {
		removeImages = compose.RemoveImagesAll
	} else {
		removeImages = compose.RemoveImagesLocal
	}

	err := c.Down(ctx, compose.RemoveOrphans(true), removeImages)
	if err != nil {
		return eris.Wrap(err, "failed to tear down")
	}

	return nil
}

type DockerComposeExporter struct {
	compose.ComposeStack
	exporterService string
	startupTimeout  time.Duration
}

// Start wait for the service to be ready and returns the endpoint.
func (e *DockerComposeExporter) Start(ctx context.Context) (string, error) {
	container, err := e.ServiceContainer(ctx, e.exporterService)
	if err != nil {
		return "", eris.Wrap(err, "failed to get service container")
	}

	strategy := wait.ForExposedPort().WithStartupTimeout(e.startupTimeout)
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

func NewDockerComposeExporter(composeStack compose.ComposeStack, exporterService string, startupTimeout time.Duration) *DockerComposeExporter {
	return &DockerComposeExporter{
		ComposeStack:    composeStack,
		exporterService: exporterService,
		startupTimeout:  startupTimeout,
	}
}
