FROM golang:1.9-alpine

LABEL maintainer="spiros.michalakopoulos@atos.net"

RUN mkdir /app
WORKDIR /app

# Install git
RUN apk add --update --no-cache git

# Get torque exporter
RUN go get github.com/spiros-atos/torque_exporter

# Install torque exporter
RUN /bin/sh $GOPATH/src/github.com/spiros-atos/torque_exporter/utils/install.sh

EXPOSE 9100

#ENTRYPOINT ["torque_exporter", "-listen-address", ":9100"]
CMD ["torque_exporter", "-host=hazelhen.hww.de", "-ssh-user=xeuspimi", "-ssh-password=F1rst0n£01"]
