#!/bin/bash
WORLD=$(curl http://metadata/computeMetadata/v1/instance/attributes/world -H "Metadata-Flavor: Google")
GCSPATH="gs://sinmetalcraft-minecraft-world/"
TAR=".tar"
cd /home/minecraft
WORLD_PATH=$GCSPATH$WORLD$TAR
sudo gsutil -o GSUtil:parallel_composite_upload_threshold=1024M cp $WORLD_PATH world.tar
sudo tar xvf world.tar
sudo rm world.tar
screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar minecraft_server.1.8.8.jar nogui