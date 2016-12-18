#!/bin/bash
cd /home/minecraft
sudo /usr/share/google/safe_format_and_mount /dev/sdb /home/minecraft/world/
sudo rm world/session.lock
WORLD=$(curl http://metadata/computeMetadata/v1/instance/attributes/world -H "Metadata-Flavor: Google")
MC_VERSION=$(curl http://metadata/computeMetadata/v1/instance/attributes/minecraft-version -H "Metadata-Flavor: Google")
sudo gsutil cp "gs://sinmetalcraft-overviewer/client/*" /home/minecraft
sudo echo 'worlds["'$WORLD'"] = "/home/minecraft/world"' >> minecraft-overviwer.config
sudo echo 'renders["normalrender"] = {' >> minecraft-overviwer.config
sudo echo '"world": "'$WORLD'",' >> minecraft-overviwer.config
sudo echo '"title": "'$WORLD'",' >> minecraft-overviwer.config
sudo echo '}' >> minecraft-overviwer.config
sudo echo 'texturepath = "/home/minecraft/minecraft_client.'$MC_VERSION.jar'"' >> minecraft-overviwer.config
sudo echo 'outputdir = "/home/minecraft/overviewer/'$WORLD'"' >> minecraft-overviwer.config

sudo overviewer.py --config=/home/minecraft/minecraft-overviwer.config
gsutil -m -h "Cache-Control: public,max-age=3600" cp -a public-read -r overviewer/$WORLD gs://sinmetalcraft-overviewer

INSTANCE_NAME=$(curl http://metadata/computeMetadata/v1/instance/hostname -H "Metadata-Flavor: Google")
INSTANCE_ZONE=$(curl http://metadata/computeMetadata/v1/instance/zone -H "Metadata-Flavor: Google")
IFS='.'
set -- $INSTANCE_NAME
INSTANCE_NAME=$(echo $1)
echo $INSTANCE_NAME

IFS='/'
set -- $INSTANCE_ZONE
INSTANCE_ZONE=$4
echo $INSTANCE_ZONE
yes | gcloud compute instances delete $INSTANCE_NAME --zone $INSTANCE_ZONE
