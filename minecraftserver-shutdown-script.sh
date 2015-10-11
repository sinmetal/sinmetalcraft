#!/bin/bash
screen -r mcs -X stuff '/save-all\n/save-off\n'
cd /home/minecraft
sudo tar cvf world.tar world
WORLD=$(curl http://metadata/computeMetadata/v1/instance/attributes/world -H "Metadata-Flavor: Google")
GCSPATH="gs://sinmetalcraft-minecraft-world/"
TAR=".tar"
WORLD_PATH=$GCSPATH$WORLD$TAR
/usr/local/bin/gsutil cp world.tar $WORLD_PATH
screen -r mcs -X stuff '/save-on\n'
sudo rm world.tar