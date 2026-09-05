#!/usr/bin/env bash
set -e

echo "===> Go kompilyatorini tekshirish..."
if ! command -v go &> /dev/null; then
    echo "===> Go topilmadi, Go 1.22 yuklab olinmoqda..."
    curl -fsSL https://go.dev/dl/go1.22.6.linux-amd64.tar.gz | tar -xz -C /tmp
    export PATH=/tmp/go/bin:$PATH
fi

echo "===> Go versiyasi:"
go version

echo "===> Kinobot Go binar faylini yig'ish..."
CGO_ENABLED=0 go build -ldflags="-s -w" -o kinobot .

echo "===> Muvaffaqiyatli yig'ildi!"
ls -lh kinobot
