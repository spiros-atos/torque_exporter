export dockerregistry="registry.test.euxdat.eu/euxdat"
export service="torque_exporter"
docker build --rm --force-rm --tag=$service .
#docker build --no-cache --rm --force-rm --tag=$service .
docker pull $dockerregistry/$service:latest
docker run $service #-host="hazelhen.hww.de" -ssh-user="xeuspimi" -ssh-password="P1l1nh@01"
