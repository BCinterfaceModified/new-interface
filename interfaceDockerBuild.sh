 #!/bin/bash


echo "version: $1"

docker build --tag new-interface:$1 .