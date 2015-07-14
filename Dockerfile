FROM kiasaki/alpine-golang

ADD ./bin/governess /
ENTRYPOINT ["/governess"]
