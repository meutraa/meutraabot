#!/usr/bin/env sh
CGO_ENABLED=0 go build -ldflags "-extldflags -static" ./cmd/meutraabot || exit
scp meutraabot moon:
ssh root@192.168.1.102 /usr/bin/env sh << EOF
        systemctl stop meutraabot && \
        cp meutraabot /etc/nixos/bin/meutraabot/ && \
        systemctl start meutraabot && \
        journalctl -xefu meutraabot
EOF
