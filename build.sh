#!/bin/bash

# 确保目标目录存在
mkdir -p lib bin

# 为 Linux 编译
GOOS=linux GOARCH=amd64 go build -o lib/back_save_liunx main.go

# 检查编译是否成功
if [ $? -ne 0 ]; then
    echo "编译失败"
    exit 1
fi

# 为 macOS 编译
# GOOS=darwin GOARCH=amd64 go build -o lib/back_save_mac main.go

# 为 Windows 编译
# GOOS=windows GOARCH=amd64 go build -o lib/back_save_windows.exe sftp_download.go

echo "构建成功！"
echo "Linux/macOS 用户: 可通过 bin/run_sftp_download.sh 运行程序"
# echo "Windows 用户: 可通过 bin/run_sftp_download.bat 运行程序"