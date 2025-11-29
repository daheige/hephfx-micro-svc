#!/usr/bin/env bash
root_dir=$(cd "$(dirname "$0")"; cd ..; pwd)

# 运行二进制文件
cd $root_dir
./main
