#!/usr/bin/env sh

echo "Starting script create"
# script -q -c "cloudflared tunnel login 2>&1 | tee /app/output.log" &
cloudflared tunnel create edgebox 2>&1 | tee /home/system/components/edgeboxctl/scripts/output.log
echo "sleeping 5 seconds"
sleep 5