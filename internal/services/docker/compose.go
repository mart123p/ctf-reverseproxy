package docker

import (
	"fmt"
	"log"
	"strings"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"github.com/mart123p/ctf-reverseproxy/internal/config"
)

type composeFile struct {
	ctfNetwork   string
	ctfNetworkId string
	mainService  int
	project      *types.Project
}

const ctfReverseProxyAnnotation = "ctf-reverseproxy"

func (d *DockerService) validation() {
	filename := config.GetString(config.CDockerComposeFile)
	workDir := config.GetString(config.CDockerComposeWorkdir)

	log.Printf("[Docker] [Compose] -> Validating compose file \"%s\" in workdir \"%s\"", filename, workDir)

	options, err := cli.NewProjectOptions([]string{filename},
		cli.WithName("ctf-challenge"),
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
						log.Fatalf("[Docker] [Compose] -> Multiple services with \"%s\" annotation found. Only one service can use the annotation", ctfReverseProxyAnnotation)
					}
					annotationFound = true
					mainService = service.Name
					d.compose.mainService = i

					//Check if a port is exposed
					if service.Expose == nil || len(service.Expose) == 0 {
						log.Fatalf("[Docker] [Compose] -> Service \"%s\" has no ports exposed. Please use the expose directive", service.Name)
					}

					if len(service.Expose) > 1 {
						log.Printf("[Docker] [Compose] -> Service \"%s\" has multiple ports exposed. Only the first port will be used", service.Name)
					}
				}
			}
		}

		//Check that no ports are exposed by the ports tag.
		if service.Ports != nil {
			log.Fatalf("[Docker] [Compose] -> Service \"%s\" has ports exposed. Please use the expose directive instead", service.Name)
		}

		//Check that an image is present
		if service.Image == "" {
			log.Fatalf("[Docker] [Compose] -> Service \"%s\" has no image specified. An image needs to be specified", service.Name)
		}

		//Check that no volumes are specified
		if service.Volumes != nil {
			log.Fatalf("[Docker] [Compose] -> Service \"%s\" has volumes specified. Volumes are not supported", service.Name)
		}
	}

	if project.Volumes != nil && len(project.Volumes) > 0 {
		log.Fatalf("[Docker] [Compose] -> Volumes are not supported")
	}

	log.Printf("[Docker] [Compose] -> Main service found: \"%s\"", mainService)
	log.Printf("[Docker] [Compose] -> Compose file validated")
	d.compose.project = project
}

func (d *DockerService) getAddr(ctfId int) string {
	return fmt.Sprintf("%s:%s", d.compose.project.Services[d.compose.mainService].Name, d.compose.project.Services[d.compose.mainService].Expose[0])
}
