# Governess

Governess is responsible for monitoring running docker containers and
triggering a restart when it receives an event indicating that a new version of
the container is available.

Events are received via [NSQ](http://nsq.io/) and should serialize into a governess.Event object.

## Configuration

Governess requires the following information:

  * URI to NSQLookupd (default: nslookupd:4161)
  * NSQ Topic where it will receive messages (default: quay)
  * URI to Docker (default: unix:///var/run/docker.sock)
  * Tag to track for all images pushed to Docker repositories (default: latest)

## Running

You can run Governess by herself from the command line or in a Docker container.
Either way, Governess requires a copy of Etcd2 to be running (as this is her
datastore, and it is used to acquire restart locks for services). It also
requires NSQ and something that sends events to NSQ. In our case, this is a
[webhooks](http://github.com/opsee/webhooks) service.

For example, assuming you are using [docker-machine](https://docs.docker.com/machine/)
with a machine named "dev":

```
eval "$(docker-machine env dev)"
export DOCKER_IP=$(docker-machine ip dev)

# Start etcd2
docker run -p 2379:2379 -d --name etcd quay.io/coreos/etcd \
     --listen-client-urls='http://0.0.0.0:2379' \
     --advertise-client-urls="http://$DOCKER_IP:2379"

# Finally, start Governess.
docker run -e ETCD_HOST=${DOCKER_IP}:2379 -d --name governess \
     --volume /var/run/docker.sock:/var/run/docker.sock \
     quay.io/opsee/governess \
     --nsqd nsqlookupd.mydomain.com:4161 \
     --topic deploy
```

## License

See [License](LICENSE.md)
