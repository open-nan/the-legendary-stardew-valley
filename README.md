# 欢迎来到传说中的云存档备份工具

这是一个全自动的存档备份工具，它基于GitHub的CI/CD功能，每天UTC时间0点运行（北京时间8点）自动通过SFTP从服务器上拉取指定文件记录到Save目录。

目前他服务于我搭建的《传说中的星露谷》云服务器

## 功能

- 拉取指定服务器上的文件，前提你开启了SFTP并拥有访问授权限
- 基于GitHub的CI/CD功能，每天UTC时间0点运行（北京时间8点），也可以手动触发
- 将存档同步到当前GitHub仓库的Save目录

## 存档结构

存档会拉取到当前项目的Save文件夹下并以当天日期命名，例如：2001-08-01。每个存档都是一个zip文件，文件名就是日期。
你可以直接通过日期来访问并下载对应日期的存档

## 如果你想fork这个项目并自己运行

你需要在你的GitHub仓库的Settings中配置以下Actions secrets：

- `SFTP_HOST`：SFTP服务器主机名
- `SFTP_PORT`：SFTP服务器端口号
- `SFTP_USER`：SFTP服务器用户名
- `SFTP_PASSWORD`：SFTP服务器密码
- `SAVE_DIR`：星露谷存档的服务器目录

## 你也可以通过git clone这个项目到本地

```bash
git clone https://github.com/open-nan/the-legendary-stardew-valley.git
```

目前我只构建了liunx版本的SFTP拉取工具，如果你想你可以自己通过golang的`go build`命令构建你自己的本地版本

```bash
go build -o main.go
```

在运行前你要准备好以下环境变量

- `SFTP_HOST`：SFTP服务器主机名
- `SFTP_PORT`：SFTP服务器端口号
- `SFTP_USER`：SFTP服务器用户名
- `SFTP_PASSWORD`：SFTP服务器密码
- `SAVE_DIR`：星露谷存档的服务器目录

## 许可证

本项目基于MIT许可证开源。