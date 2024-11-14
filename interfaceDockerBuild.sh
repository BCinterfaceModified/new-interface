 #!/bin/bash


echo "version: $1"

docker build --tag bcinterface:$1 .