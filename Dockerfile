FROM kiasaki/alpine-golang

ADD ./governess /
ENTRYPOINT ["/governess"]
