#!/usr/bin/env sh

echo "Starting script login"
cloudflared tunnel login 2>&1 | tee /home/system/components/edgeboxctl/scripts/output.log &
echo "sleeping 5 seconds"
sleep 5