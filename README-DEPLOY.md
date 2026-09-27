# WeComX deployment

This setup intentionally avoids Docker and Redis. The service is a single Go binary supervised by systemd.

## Build on VPS

    sudo apt-get update
    sudo apt-get install -y golang-go
    cd ~/wecomx
    ./build.sh

For the current x86_64 Oracle VPS this produces an amd64 Linux binary.

## Install

    sudo ./install-wecomchan.sh ./wecomchan
    sudo nano /etc/wecomchan.env
    sudo chmod 600 /etc/wecomchan.env
    sudo systemctl restart wecomchan

Required variables: SENDKEY, WECOM_CID, WECOM_SECRET, WECOM_AID, WECOM_TOUID.

Check: `sudo systemctl status wecomchan`, `curl http://127.0.0.1:8080/health`.

## Cloudflare Tunnel

Public path: `https://push.example.com -> Cloudflare Tunnel -> http://127.0.0.1:8080`.

No public inbound port 8080 is required.

## Security

Keep `/etc/wecomchan.env` at mode 0600. Never commit WeCom credentials. Use a long random SENDKEY. Keep port 8080 bound to localhost. Prefer Bearer authentication. Keep this VPS separate from VPN/proxy workloads when its public IP is important.
