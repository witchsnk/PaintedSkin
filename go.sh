#!/bin/sh
# 沙箱环境下全局 Go 模块缓存不可写，统一使用工作区内缓存。
# 本机 go env 里 GO111MODULE=off 是全局设置，会让 go 退回 GOPATH 模式，
# 导致本仓库的 module PaintedSkin 解析失败（cannot find package "PaintedSkin/..."）。
# 这里强制打开 module 模式，使 ./go.sh 可独立于调用者的全局 go env 工作。
# 用法: ./go.sh test ./...   （等价于 go <args>）
DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
GO111MODULE=on \
GOFLAGS=-mod=mod GOSUMDB=off \
GOMODCACHE="$DIR/.gomodcache" \
GOCACHE="$DIR/.gocache" \
exec go "$@"
