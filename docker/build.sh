#!/bin/bash

export IMAGE_NAME=torque_exporter
#docker build --rm=true -t $IMAGE_NAME /home/linux/ATOSES_spiros/GITHUB-torque_exporter/torque_exporter/docker
docker build --no-cache --rm=true -t $IMAGE_NAME /home/linux/ATOSES_spiros/GITHUB-torque_exporter/torque_exporter/docker
docker tag $IMAGE_NAME registry.test.euxdat.eu/euxdat/torque_exporter
docker push registry.test.euxdat.eu/euxdat/torque_exporter

