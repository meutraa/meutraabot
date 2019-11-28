#!/usr/bin/env sh
go build ./cmd/meutraabot || exit

rm meutraabot meutraa-leaderboard meutraa-chat

code=$(find -name "*.go" -exec cat {} \; | sed -r '/^\s*$/d')
lines=$(echo "${code}" | wc -l)
words=$(echo "${code}" | wc -w)
chars=$(echo "${code}" | wc -c)

sed -i -e "s/Lines:[^,]*/Lines:${lines}/" \
        -e "s/Words:[^,]*/Words:${words}/" \
        -e "s/Characters:[^,]*/Characters:${chars}/" \
        cmd/meutraabot/modules/management/management.go

rsync -aP -e 'ssh -p 2020' --delete ./ lost.host:meutraabot/
ssh -p 2020 lost.host /usr/bin/env sh << EOF
cd meutraabot
go build ./cmd/meutraabot && \
        sudo systemctl stop meutraabot && \
        sudo cp meutraabot /etc/nixos/bin/ && \
        sudo systemctl start meutraabot && \
        sudo journalctl -xefu meutraabot
EOF
