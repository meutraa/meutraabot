#!/usr/bin/env sh
go build ./cmd/meutraa-widgets || exit

rm meutraabot meutraa-widgets

rsync -aP -e 'ssh -p 2020' --delete ./ lost.host:meutraabot/ && \
ssh -p 2020 lost.host /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraa-widgets && \
        sudo systemctl stop meutraa-widgets && \
        sudo cp meutraa-widgets /etc/nixos/bin/ && \
        sudo systemctl start meutraa-widgets && \
        sudo journalctl -xefu meutraa-widgets
EOF
