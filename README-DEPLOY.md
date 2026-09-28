# WeComX deployment

WeComX is derived from the open-source wecomchan project by easychen:
https://github.com/easychen/wecomchan

## Build on the VPS

    git clone https://github.com/niceyayale/wecomx.git
    cd wecomx
    sudo apt-get update
    sudo apt-get install -y golang-go
    chmod +x build.sh install-wecomchan.sh
    ./build.sh

## Configure

Create /etc/wecomchan.env with the real enterprise credentials. Keep this file private:

    sudo chmod 600 /etc/wecomchan.env

Do not put real SENDKEY, WECOM_SECRET, callback Token, callback EncodingAESKey, or access tokens in Git.

## Install

    sudo ./install-wecomchan.sh ./wecomchan
    sudo systemctl restart wecomchan

## Public callback

The WeCom callback endpoint is:

    http://YOUR_PUBLIC_IP:28473/hook_path

Configure it in the WeCom application as required by the standard callback verification flow.

## Cloudflare Tunnel

Recommended public sender endpoint:

    https://push.example.com/wecomchan
        -> Cloudflare Tunnel
        -> http://127.0.0.1:8080
        -> WeComX

No public inbound 8080 is required.

TCP 28473 is separate from the push API and can remain enabled for the WeCom callback.

## Runtime hardening

The service runs as the unprivileged wecomchan account with systemd sandboxing.

The callback listener uses:
- 16 KB maximum HTTP headers
- 128 KB maximum callback body
- strict read/write/idle timeouts
- timestamp validation with a ±5 minute window
- constant-time signature comparison

The callback endpoint performs verification/decryption only and has no shell, file upload, database, or command execution interface.
