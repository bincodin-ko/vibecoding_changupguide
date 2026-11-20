#!/bin/sh
# Windows용 실행 파일(.exe)을 만든다. 리눅스·맥에서도 그대로 돌아간다.
#   sh build.sh        → ../dist/VibeCoach.exe
set -e
cd "$(dirname "$0")"
go vet ./...
go test ./...
mkdir -p ../dist
# -H windowsgui: 검은 명령 창 없이 프로그램 창만 뜨게 한다.
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-H windowsgui -s -w" -o ../dist/VibeCoach.exe .
ls -l ../dist/VibeCoach.exe
