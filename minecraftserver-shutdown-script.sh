#!/bin/bash
sudo screen -r -X stuff '/stop\n'
cd /home/minecraft
sudo tar cvf world.tar world
WORLD=$(curl http://metadata/computeMetadata/v1/instance/attributes/world -H "Metadata-Flavor: Google")
GCSPATH="gs://sinmetalcraft-minecraft-world-dra/"
TAR=".tar"
WORLD_PATH=$GCSPATH$WORLD$TAR
/usr/local/bin/gsutil cp world.tar $WORLD_PATH
sudo rm world.tar