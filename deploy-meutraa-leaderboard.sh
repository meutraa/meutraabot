#!/usr/bin/env sh
go build ./cmd/meutraa-leaderboard || exit

rm meutraabot meutraa-leaderboard meutraa-chat

rsync -aP -e 'ssh -p 2020' --delete ./ lost.host:meutraabot/
ssh -p 2020 lost.host /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraabot-leaderboard && \
        sudo systemctl stop meutraabot-leaderboard && \
        sudo cp meutraabot-leaderboard /etc/nixos/bin/ && \
        sudo systemctl start meutraabot-leaderboard && \
        sudo journalctl -xefu meutraabot-leaderboard
EOF
