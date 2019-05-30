export dockerregistry="registry.test.euxdat.eu/euxdat"
export service="torque_exporter"
#export input_folder=$(pwd)/input_test
#export output_folder=$(pwd)/output
docker build --rm --force-rm --tag=$service .
docker pull $dockerregistry/$service:latest
#docker run -v "$input_folder:/var/data" \
#  -v "$output_folder:/var/output" \
#  -e INPUT_RASTERFILE=raster_4326.tif \
#  -e INPUT_SHAPEFILE=vector_4326.shp \
#  -e OUTPUT_RASTERFILE=raster_4326_output.tif \
#  -e OUTPUT_STATISTICSFILE=statistics.json \
#  $service
#docker rmi -f $service
docker run $service -host="hazelhen.hww.de" -ssh-user="xeuspimi" -ssh-password="P1l1nh@01"

