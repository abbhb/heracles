package core

import (
	"context"

	"github.com/rotisserie/eris"
	"github.com/testcontainers/testcontainers-go/modules/compose"
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
