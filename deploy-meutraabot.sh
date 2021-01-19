#!/usr/bin/env sh
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-extldflags -static" ./cmd/meutraabot || exit
rsync -aP meutraabot 192.168.1.102:
ssh root@192.168.1.102 /usr/bin/env sh << EOF
        systemctl stop meutraabot && \
        cp meutraabot /etc/nixos/bin/meutraabot/ && \
        systemctl start meutraabot && \
        journalctl -xefu meutraabot
EOF
