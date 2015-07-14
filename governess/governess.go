package governess

import (
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
	"github.com/opsee/governess/messaging"
	"github.com/samalba/dockerclient"
)

var (
	logger = logging.MustGetLogger("governess")
)

/*
{
  "repository": "mynamespace/repository",
  "namespace": "mynamespace",
  "name": "repository",
  "docker_url": "quay.io/mynamespace/repository",
  "homepage": "https://quay.io/repository/mynamespace/repository",
  "visibility": "public",
  "updated_tags": {
    "latest": "b750fe79269d2ec9a3c593ef05b4332b1d1a02a62b4accb2c21d589ff2f5f2dc"
  },
  "pushed_image_count": 10,
  "pruned_image_count": 3
}
*/
type RepositoryPush struct {
	Repository       string            `json:"string"`
	Name             string            `json:"name"`
	Namespace        string            `json:"namespace"`
	DockerURL        string            `json:"docker_url"`
	Homepage         url.URL           `json:"homepage"`
	Visibility       string            `json:"visibility"`
	UpdatedTags      map[string]string `json:"updated_tags"`
	PrunedImageCount int               `json:"pruned_image_count"`
	PushedImageCount int               `json:"pushed_image_count"`
}

type ManagedContainer struct {
	Id    string
	Image string
	Tag   string
}

// ContainerRegistry maintains the two-way mapping
type ContainerRegistry struct {
	sync.Mutex
	containersById    map[string]*ManagedContainer
	containersByImage map[string]map[*ManagedContainer]bool
}

func NewContainerRegistry() *ContainerRegistry {
	return &ContainerRegistry{
		containersById:    make(map[string]*ManagedContainer),
		containersByImage: make(map[string]map[*ManagedContainer]bool),
	}
}

func makeManagedContainer(id, origImage string) *ManagedContainer {
	parts := strings.Split(origImage, ":")
	image := parts[0]

	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}

	return &ManagedContainer{
		Id:    id,
		Image: image,
		Tag:   tag,
	}
}

func (r *ContainerRegistry) GetByImage(image string) map[*ManagedContainer]bool {
	return r.containersByImage[image]
}

func (r *ContainerRegistry) GetById(containerId string) *ManagedContainer {
	return r.containersById[containerId]
}

func (r *ContainerRegistry) Add(container *ManagedContainer) {
	r.Lock()
	defer r.Unlock()

	r.containersByImage[container.Image][container] = true
	r.containersById[container.Id] = container
}

func (r *ContainerRegistry) Remove(container *ManagedContainer) {
	r.Lock()
	defer r.Unlock()

	delete(r.containersByImage[container.Image], container)
}

type Governess struct {
	tag      string
	channel  chan string
	consumer *messaging.Consumer
	docker   *dockerclient.DockerClient
	registry *ContainerRegistry
}

func NewGoverness(tag string, consumer *messaging.Consumer, dockerURI *url.URL) *Governess {
	docker, err := dockerclient.NewDockerClient(dockerURI.String(), nil)
	if err != nil {
		panic(err)
	}
	g := &Governess{
		tag:      tag,
		consumer: consumer,
		docker:   docker,
		channel:  make(chan string, 1),
		registry: NewContainerRegistry(),
	}

	return g
}

func dockerClientCallback(e *dockerclient.Event, ec chan error, args ...interface{}) {
	registry := args[0].(*ContainerRegistry)
	dockerClient := args[1].(*dockerclient.DockerClient)
	switch e.Status {
	case "start":
		go func() {
			containerInfo, err := dockerClient.InspectContainer(e.Id)
			if err != nil {
				ec <- err
				return
			}
			container := makeManagedContainer(containerInfo.Id, containerInfo.Image)
			registry.Add(container)
		}()
	case "stop", "die", "kill":
		go func() {
			containerInfo, err := dockerClient.InspectContainer(e.Id)
			if err != nil {
				ec <- err
				return
			}
			container := makeManagedContainer(containerInfo.Id, containerInfo.Image)
			registry.Remove(container)
		}()
	}
}

func (g *Governess) syncRegistry() {
	containers, err := g.docker.ListContainers(false, false, "")
	if err != nil {
		logger.Error("Unable to list containers: ", err.Error())
	}

	for _, container := range containers {
		exists := g.registry.GetById(container.Id)
		if exists == nil {
			g.registry.Add(makeManagedContainer(container.Id, container.Image))
		}
	}
}

func (g *Governess) Govern() {
	errChan := make(chan error)
	g.docker.StartMonitorEvents(dockerClientCallback, errChan, g.registry)

	syncTicker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			<-syncTicker.C

			g.syncRegistry()
		}
	}()

	// Monitor errors from the err channel
	go func() {
		for err := range errChan {
			logger.Error("Received Docker error event: ", err)
		}
	}()

	// Monitor deploy events
	go func() {
		for event := range g.consumer.Channel() {
			if event.Type() == "RepositoryPush" {
				pushEvent := new(RepositoryPush)
				err := json.Unmarshal([]byte(event.Body()), pushEvent)
				if err != nil {
					logger.Error("Unable to deserialize message: ", event.Body())
				}

				if pushEvent.UpdatedTags[g.tag] != "" {
					dockerURL := pushEvent.DockerURL
					logger.Error("Kicking container: ", dockerURL)

					g.registry.Lock()
					for container, _ := range g.registry.GetByImage(dockerURL) {
						if container.Tag == g.tag {
							err := g.docker.StopContainer(container.Id, 5)
							if err != nil {
								logger.Error(err.Error())
							}
						}
					}
				}
			}
		}
	}()
}
