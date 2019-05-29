FROM golang:1.9-alpine

LABEL maintainer="spiros.michalakopoulos@atos.net"

RUN mkdir /app
WORKDIR /app

# Install git
RUN apk add --update --no-cache git

# Get slurm exporter
RUN go get github.com/spiros-atos/torque_exporter

# Install slurm exporter
RUN /bin/sh $GOPATH/src/github.com/spiros-atos/torque_exporter/utils/install.sh

EXPOSE 9100

ENTRYPOINT ["torque_exporter", "-listen-address", ":9100"]

