# WeComX

Lightweight personal WeCom push gateway for a 1C1G VPS.

No Redis, database, or Docker. Single Go binary, systemd, localhost-only listener, in-memory token cache, text/markdown, and GET compatibility.

## Quick start

    sudo apt-get update
    sudo apt-get install -y golang-go
    git clone https://github.com/niceyayale/wecomx.git
    cd wecomx
    ./build.sh
    sudo ./install-wecomchan.sh ./wecomchan
    sudo nano /etc/wecomchan.env
    sudo systemctl restart wecomchan

Health: `curl http://127.0.0.1:8080/health`

Recommended call:

    curl -X POST 'http://127.0.0.1:8080/wecomchan' -H 'Authorization: Bearer YOUR_SENDKEY' -H 'Content-Type: application/json' -d '{"msg_type":"markdown","msg":"**WeComX test**"}'

See README-DEPLOY.md for Cloudflare Tunnel details.
