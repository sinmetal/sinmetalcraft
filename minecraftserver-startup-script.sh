#!/bin/bash
# Update Cloud DNS
sudo gsutil cp gs://sinmetalcraft-minecraft-shell/dns.sh .
sudo chmod 700 dns.sh
sudo ./dns.sh
# Minecraft Server Start
cd /home/minecraft
sudo /usr/share/google/safe_format_and_mount /dev/sdb /home/minecraft/world/
sudo rm world/session.lock
STATE=$(curl http://metadata/computeMetadata/v1/instance/attributes/state -H "Metadata-Flavor: Google")
MC_VERSION=$(curl http://metadata/computeMetadata/v1/instance/attributes/minecraft-version -H "Metadata-Flavor: Google")
MC_APP="minecraft_server."
JAR=".jar"
MC_JAR=$MC_APP$MC_VERSION$JAR
echo $STATE
if [ ${STATE} = "exists" ]; then
  echo "EXISTS INSTNCE"
  sudo rm world/session.lock
  screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar $MC_JAR nogui
  exit 0
fi
echo "NEW INSTNCE"
screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar $MC_JAR nogui
gcloud compute instances add-metadata $HOSTNAME --zone=asia-northeast1-b --metadata state=exists
