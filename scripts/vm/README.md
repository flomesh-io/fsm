# 1. Copy the pipy program to the host.  
You need to add the path of pipy to PATH system variable.

# 2. Set the required environment variables.  
for example:    
export PIPY_NIC=eth0  

export PIPY_REPO=http://10.10.10.1:6060/repo/fsm-sidecar/sidecar.vm49.derive-vm/

export PIPY_DNS=10.10.10.1 #change to CLUSTER-IP/EXTERNAL-IP of fsm-system/fsm-controller

# 3. Run the pipy process.
sh run-sidecar.sh
