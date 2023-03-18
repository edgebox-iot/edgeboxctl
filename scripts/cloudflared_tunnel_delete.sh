#!/usr/bin/env sh

echo "Starting script delete"
TUNNEL_ORIGIN_CERT=/home/system/.cloudflared/cert.pem
cloudflared tunnel delete edgebox 2>&1 | tee /home/system/components/edgeboxctl/scripts/delete_output.log &
echo "sleeping 5 seconds"
sleep 5