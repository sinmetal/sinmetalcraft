#!/bin/bash
cd /home/minecraft
STATE=$(curl http://metadata/computeMetadata/v1/instance/attributes/state -H "Metadata-Flavor: Google")
echo $STATE
if [ ${STATE} = "exists" ]; then
  echo "EXISTS INSTNCE"
  sudo rm world/session.lock
  screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar minecraft_server.1.8.jar nogui
  exit 0
fi
echo "NEW INSTNCE"
WORLD=$(curl http://metadata/computeMetadata/v1/instance/attributes/world -H "Metadata-Flavor: Google")
GCSPATH="gs://sinmetalcraft-minecraft-world-dra/"
TAR=".tar"
WORLD_PATH=$GCSPATH$WORLD$TAR
sudo gsutil cp $WORLD_PATH world.tar
sudo tar xvf world.tar
sudo rm world.tar
sudo rm world/session.lock
screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar minecraft_server.1.8.jar nogui
gcloud compute instances add-metadata $HOSTNAME --zone=asia-east1-b --metadata state=exists
gsutil cp gs://sinmetalcraft-minecraft-shell/minecraftserver-backup.sh backup.sh
sudo chmod 755 backup.sh