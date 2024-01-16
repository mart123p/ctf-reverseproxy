package docker

import (
	"log"
	"strings"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"github.com/mart123p/ctf-reverseproxy/internal/config"
)

type composeFile struct {
	mainService int
	project     *types.Project
}

const ctfReverseProxyAnnotation = "ctf-reverseproxy"

func (d *DockerService) validation() {
	filename := config.GetString(config.CDockerComposeFile)
	workDir := config.GetString(config.CDockerComposeWorkdir)

	options, err := cli.NewProjectOptions([]string{filename},
		cli.WithWorkingDirectory(workDir),
		cli.WithDotEnv,
		cli.WithConfigFileEnv,
		cli.WithDefaultConfigPath,
	)
	if err != nil {
		log.Fatalf("[Docker] [Compose] -> Failed to configure project options, %s", err)
	}

	project, err := cli.ProjectFromOptions(options)
	if err != nil {
		log.Fatalf("[Docker] [Compose] -> Failed to load project, %s", err)
	}

	_, err = project.MarshalYAML()
	if err != nil {
		log.Fatalf("[Docker] [Compose] -> Failed to marshall project, %s", err)
	}

	annotationFound := false
	mainService := ""

	for i, service := range project.Services {

		//Check if the annotations are present
		if service.Annotations != nil {
			if annotation, ok := service.Annotations[ctfReverseProxyAnnotation]; ok {
				if strings.ToLower(annotation) == "true" {
					if annotationFound {
						log.Fatalf("[Docker] [Compose] -> Multiple services with %s annotation found. Only one service can use the annotation", ctfReverseProxyAnnotation)
					}
					annotationFound = true
					mainService = service.Name
					d.compose.mainService = i
				}
			}
		}

		//Check that no ports are exposed by the ports tag.
		if service.Ports != nil {
			log.Fatalf("[Docker] [Compose] -> Service %s has ports exposed. Please use the expose directive instead", service.Name)
		}
	}

	log.Printf("[Docker] [Compose] -> Main service found: %s", mainService)
	log.Printf("[Docker] [Compose] -> Compose file validated")
	d.compose.project = project
}
