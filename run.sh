#!/bin/bash
set -e

# cd /tiflash/monitor-9100/scripts
# ./supervisor_blackbox_exp.sh >supervisor_blackbox_exp.log 2>&1 & 
# ./supervisor_node_exp.sh >supervisor_node_exp.log 2>&1 & 
cd /tiflash
exec /tiflash/rpc_server

