# WeComX

A lightweight personal WeCom (企业微信) push gateway derived from the open-source wecomchan project by easychen:
https://github.com/easychen/wecomchan

WeComX is a second-generation personal adaptation focused on small VPS deployments, minimal dependencies, and long-running self-hosted use.

## Design

- Single Go binary
- No Redis
- No database
- No Docker
- systemd supervised
- Main push API listens only on 127.0.0.1:8080
- Optional public WeCom callback listener on 0.0.0.0:28473/hook_path
- In-memory access_token cache
- Text and markdown push
- Bearer or X-API-Key authentication
- GET compatibility with the original WeComChan-style sender
- Callback signature and timestamp validation
- Bounded HTTP headers and callback request body
- systemd sandboxing with a non-root service account

## Build

Compile directly on the target VPS:

    git clone https://github.com/niceyayale/wecomx.git
    cd wecomx
    sudo apt-get update
    sudo apt-get install -y golang-go
    chmod +x build.sh install-wecomchan.sh
    ./build.sh

## Configure

Create /etc/wecomchan.env with real values. Never commit real credentials.

    SENDKEY=replace-with-a-long-random-private-key
    WECOM_CID=replace-with-your-enterprise-id
    WECOM_SECRET=replace-with-your-application-secret
    WECOM_AID=replace-with-your-application-agent-id
    WECOM_TOUID=@all

Optional callback:

    WECOM_CALLBACK_ENABLED=true
    WECOM_CALLBACK_ADDR=0.0.0.0:28473
    WECOM_CALLBACK_TOKEN=replace-with-wecom-callback-token
    WECOM_CALLBACK_AES_KEY=replace-with-wecom-callback-encoding-aes-key

## Install

    sudo ./install-wecomchan.sh ./wecomchan
    sudo chmod 600 /etc/wecomchan.env
    sudo systemctl restart wecomchan

Health:

    curl http://127.0.0.1:8080/health

Push:

    curl -X POST 'https://push.example.com/wecomchan' -H 'Authorization: Bearer YOUR_SENDKEY' -H 'Content-Type: application/json' -d '{"msg_type":"text","msg":"WeComX test"}'

## Callback

The callback endpoint is:

    0.0.0.0:28473/hook_path

It implements the standard WeCom GET verification flow and a minimal POST callback acknowledgement.

Keep 28473 protected by the cloud firewall and host firewall. Do not expose 8080 directly when using Cloudflare Tunnel.

## Upstream attribution

WeComX is a derivative / second-generation adaptation of easychen/wecomchan and keeps the upstream project clearly credited.

Upstream:
https://github.com/easychen/wecomchan

WeComX modifications include the localhost-only push listener, callback support, access token in-memory caching, bounded HTTP input, callback timestamp validation, constant-time signature comparison, and systemd sandboxing.
