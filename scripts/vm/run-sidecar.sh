#!/bin/bash

if [ "$(id -u)" -ne 0 ]
then
    echo "Please run as root." >&2
    exit 1
fi

if [ "x$(id -u pipy 2>/dev/null)" != "x1500" ]
then
    useradd --no-create-home -r pipy -u 1500 -s /usr/sbin/nologin
    if [ "x$(id -u pipy 2>/dev/null)" != "x1500" ]
    then
        echo "Unable to create user pipy"
        exit 1
    fi
fi

if [ -z "$PIPY_NIC" ]
then
      echo "Please set the PIPY_NIC environment variable, for example: PIPY_NIC=eth0"
      exit 1
fi

if [ -z "$PIPY_DNS" ]
then
      echo "Please set the PIPY_DNS environment variable, for example: PIPY_DNS=8.8.8.8"
      exit 1
fi

ip=$(ip -4 addr show "$PIPY_NIC" 2>/dev/null | grep inet | sed 's/\// /g' | awk '{print $2}')
if [ -z "$ip" ]
then
    echo  "Unable to get ip from nic [$PIPY_NIC]"
    exit 1
fi

if [ -z "$PIPY_REPO" ]
then
      echo "Please set the PIPY_REPO environment variable"
      exit 1
fi

if [ "$(
iptables-restore <<EOF
*filter
:INPUT ACCEPT [0:0]
:FORWARD ACCEPT [0:0]
:OUTPUT ACCEPT [0:0]
-A INPUT -m state --state ESTABLISHED -j ACCEPT
COMMIT
*nat
:FSM_PROXY_INBOUND - [0:0]
:FSM_PROXY_IN_REDIRECT - [0:0]
:FSM_PROXY_OUTBOUND - [0:0]
:FSM_PROXY_OUT_REDIRECT - [0:0]
-A FSM_PROXY_IN_REDIRECT -p tcp -j REDIRECT --to-port 15003
-A PREROUTING -p tcp -j FSM_PROXY_INBOUND
-A FSM_PROXY_INBOUND -p tcp -m multiport --dports 22 -j RETURN
-A FSM_PROXY_INBOUND -p tcp --dport 15010 -j RETURN
-A FSM_PROXY_INBOUND -p tcp --dport 15901 -j RETURN
-A FSM_PROXY_INBOUND -p tcp --dport 15902 -j RETURN
-A FSM_PROXY_INBOUND -p tcp --dport 15903 -j RETURN
-A FSM_PROXY_INBOUND -p tcp --dport 15904 -j RETURN
-A FSM_PROXY_INBOUND -p tcp -j FSM_PROXY_IN_REDIRECT
-A FSM_PROXY_OUT_REDIRECT -p tcp -j REDIRECT --to-port 15001
-A FSM_PROXY_OUT_REDIRECT -p tcp --dport 15000 -j ACCEPT
-A OUTPUT -p tcp -j FSM_PROXY_OUTBOUND
-A FSM_PROXY_OUTBOUND -d $ip/32 -m owner --uid-owner 1500 -j RETURN
-A FSM_PROXY_OUTBOUND -o lo ! -d 127.0.0.1/32 -m owner --uid-owner 1500 -j FSM_PROXY_IN_REDIRECT
-A FSM_PROXY_OUTBOUND -o lo -m owner ! --uid-owner 1500 -j RETURN
-A FSM_PROXY_OUTBOUND -m owner --uid-owner 1500 -j RETURN
-A FSM_PROXY_OUTBOUND -d 127.0.0.1/32 -j RETURN
-A FSM_PROXY_OUTBOUND -j FSM_PROXY_OUT_REDIRECT
COMMIT
EOF
)" ];
then
    echo "Unable to set iptables."
    exit 1
fi

if ! grep "$PIPY_DNS" /etc/resolv.conf >/dev/null
then
    sed -i "0,/^nameserver/!b;//i\nameserver $PIPY_DNS" /etc/resolv.conf
fi

if ! grep '^search svc.cluster.local' /etc/resolv.conf >/dev/null
then
    sed -i '0,/^search/{s/search/search svc.cluster.local cluster.local/}' /etc/resolv.conf
fi

ns=$(grep "^nameserver" /etc/resolv.conf | grep -v "$PIPY_DNS" | head -n 1 | awk '{print $2}')
if [ -n "$ns" ]
then
    # export PIPY_NAMESERVER=$ns
    echo ""
fi

chmod 755 ./pipy

nohup runuser -u pipy -- ./pipy --admin-port=6060 "$PIPY_REPO#?ip=$ip" &

