#!/bin/bash
# Update Cloud DNS
sudo gsutil cp gs://sinmetalcraft-minecraft-shell/dns.sh .
sudo chmod 700 dns.sh
sudo ./dns.sh
# Minecraft Server Start
cd /home/minecraft
sudo gsutil cp gs://sinmetalcraft-minecraft-shell/ops.json .
WORLD=$(curl http://metadata/computeMetadata/v1/instance/attributes/world -H "Metadata-Flavor: Google")
WORLD_DISK_PREFIX=/dev/disk/by-id/google-minecraft-world-
WORLD_DISK=$WORLD_DISK_PREFIX$WORLD
sudo mount -o discard,defaults $WORLD_DISK /home/minecraft/world/
sudo rm world/session.lock
MC_VERSION=$(curl http://metadata/computeMetadata/v1/instance/attributes/minecraft-version -H "Metadata-Flavor: Google")
MC_APP="minecraft_server."
JAR=".jar"
MC_JAR=$MC_APP$MC_VERSION$JAR
GCS_BUCKET=gs://sinmetalcraft-minecraft-jar/
GCS_MC_JAR_PATH=$GCS_BUCKET$MC_JAR
sudo gsutil cp $GCS_MC_JAR_PATH .
STATE=$(curl http://metadata/computeMetadata/v1/instance/attributes/state -H "Metadata-Flavor: Google")
echo $STATE
if [ ${STATE} = "exists" ]; then
  echo "EXISTS INSTNCE"
  sudo rm world/session.lock
  sudo screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar $MC_JAR nogui
  exit 0
fi
echo "NEW INSTNCE"
sudo screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar $MC_JAR nogui
gcloud compute instances add-metadata $HOSTNAME --zone=asia-northeast1-b --metadata state=exists
