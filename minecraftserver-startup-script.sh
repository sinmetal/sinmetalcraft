#!/bin/bash
cd /home/minecraft
sudo /usr/share/google/safe_format_and_mount /dev/sdb /home/minecraft/world/
sudo rm world/session.lock
STATE=$(curl http://metadata/computeMetadata/v1/instance/attributes/state -H "Metadata-Flavor: Google")
echo $STATE
if [ ${STATE} = "exists" ]; then
  echo "EXISTS INSTNCE"
  sudo rm world/session.lock
  screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar minecraft_server.1.8.jar nogui
  exit 0
fi
echo "NEW INSTNCE"
screen -d -m -S mcs java -Xms1G -Xmx7G -d64 -jar minecraft_server.1.8.jar nogui
gcloud compute instances add-metadata $HOSTNAME --zone=asia-east1-b --metadata state=exists
