# Torque Prometheus Exporter Docker image

Prometheus exporter that publish Torque metrics from a remote HPC (connected by ssh).  

## Usage

```
# docker run --rm -d -p 9100:9100 spiros-atos/torque_exporter -host=<HOST> -ssh-user=<USER> -ssh-password=<PASSWD>
ea994b6b6ac2c73f10ca2a1150e32938031ad98a786dab5554772140c1a35c16

# docker ps -a
CONTAINER ID        IMAGE                   COMMAND                  CREATED         STATUS              PORTS                     NAMES
ea994b6b6ac2        spiros-atos/torque_exporter   "torque_exporter -l..."   7 minutes ago   Up 3 seconds        0.0.0.0:9100->9100/tcp   dreamy_spence

$ curl localhost:9100/metrics
# HELP ....
```
