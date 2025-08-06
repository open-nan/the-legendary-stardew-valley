package main

import (
	"archive/zip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// 配置结构体
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	SaveDir  string
}

func loadConfig() (*Config, error) {
	// 从环境变量获取配置
	host := os.Getenv("SFTP_HOST")
	if host == "" {
		return nil, fmt.Errorf("缺少必要的环境变量，请设置 SFTP_HOST")
	}
	portStr := os.Getenv("SFTP_PORT")
	if portStr == "" {
		return nil, fmt.Errorf("缺少必要的环境变量，请设置 SFTP_PORT")
	}
	user := os.Getenv("SFTP_USER")
	if user == "" {
		return nil, fmt.Errorf("缺少必要的环境变量，请设置 SFTP_USER")
	}
	password := os.Getenv("SFTP_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("缺少必要的环境变量，请设置 SFTP_PASSWORD")
	}
	saveDir := os.Getenv("SAVE_DIR")
	if saveDir == "" {
		return nil, fmt.Errorf("缺少必要的环境变量，请设置 SAVE_DIR")
	}

	// 转换端口字符串为整数
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("无效的端口号: %w", err)
	}

	// 创建并返回配置对象
	return &Config{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		SaveDir:  saveDir,
	}, nil
}

// 创建SFTP客户端
func createSFTPClient(user, password, host string, port int) (*sftp.Client, error) {
	// 创建SSH配置
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// 注意：在生产环境中应该验证主机密钥
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 建立SSH连接
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("无法建立SSH连接: %w", err)
	}

	// 创建SFTP客户端
	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("无法创建SFTP客户端: %w", err)
	}

	return client, nil
}

// 从SFTP服务器下载文件
func downloadFile(client *sftp.Client, remotePath, localPath string) error {
	// 打开远程文件
	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("无法打开远程文件: %w", err)
	}
	defer remoteFile.Close()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("无法创建本地文件: %w", err)
	}
	defer localFile.Close()

	// 复制文件内容
	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		return fmt.Errorf("文件复制失败: %w", err)
	}

	return nil
}

// 递归下载目录及其内容
func downloadDirectory(client *sftp.Client, remoteDir, localDir string) error {
	// 创建本地目录
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("无法创建本地目录: %w", err)
	}

	// 列出远程目录中的所有条目
	entries, err := client.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("无法读取远程目录: %w", err)
	}

	for _, entry := range entries {
		remotePath := filepath.Join(remoteDir, entry.Name())
		localPath := filepath.Join(localDir, entry.Name())

		if entry.IsDir() {
			// 如果是目录，递归下载
			if err := downloadDirectory(client, remotePath, localPath); err != nil {
				return fmt.Errorf("下载目录 %s 失败: %w", remotePath, err)
			}
		} else {
			// 如果是文件，直接下载
			if err := downloadFile(client, remotePath, localPath); err != nil {
				return fmt.Errorf("下载文件 %s 失败: %w", remotePath, err)
			}
			log.Printf("已下载: %s -> %s", remotePath, localPath)
		}
	}

	return nil
}

// 压缩目录到ZIP文件
func zipDirectory(source, target string) error {
	// 创建ZIP文件
	zipFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("无法创建ZIP文件: %w", err)
	}
	defer zipFile.Close()

	// 创建ZIP写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历目录并添加文件到ZIP
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 创建ZIP文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 设置压缩方法
		header.Method = zip.Deflate

		// 计算相对路径
		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header.Name = relativePath

		// 如果是目录，添加斜杠
		if info.IsDir() {
			header.Name += "/"
		}

		// 创建ZIP条目
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("压缩目录失败: %w", err)
	}

	return nil
}

// 删除目录及其内容
func removeDirectory(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("删除目录失败: %w", err)
	}
	return nil
}

// 计算文件的MD5值
func calculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件: %w", err)
	}
	defer file.Close()

	hm := md5.New()
	if _, err := io.Copy(hm, file); err != nil {
		return "", fmt.Errorf("计算MD5失败: %w", err)
	}

	return hex.EncodeToString(hm.Sum(nil)), nil
}

// 递归计算目录中所有文件的MD5并写入文件
func generateMD5File(rootDir, outputFile string) error {
	// 创建MD5文件
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("无法创建MD5文件: %w", err)
	}
	defer file.Close()

	// 遍历目录
	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// 计算MD5
		md5Sum, err := calculateFileMD5(path)
		if err != nil {
			log.Printf("计算文件 %s 的MD5失败: %v", relPath, err)
			return nil // 继续处理其他文件
		}

		// 写入MD5文件 (文件名 + 空格 + MD5)
		_, err = fmt.Fprintf(file, "%-40s %s\n", relPath, md5Sum)
		if err != nil {
			return fmt.Errorf("写入MD5到文件失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("生成MD5文件失败: %w", err)
	}

	return nil
}

func main() {
	// 加载配置
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 获取当天的时间以YYYY-MM-DD的格式保存
	now := time.Now().Format("2006-01-02")

	// 创建SFTP客户端
	client, err := createSFTPClient(config.User, config.Password, config.Host, config.Port)
	if err != nil {
		log.Fatalf("创建SFTP客户端失败: %v", err)
	}
	defer client.Close()

	// 下载整个目录
	remoteDir := config.SaveDir
	localDir := fmt.Sprintf("./Save/%s", now)

	err = downloadDirectory(client, remoteDir, localDir)
	if err != nil {
		log.Fatalf("下载目录失败: %v", err)
	}

	log.Println("目录下载成功!")

	// 生成MD5文件
	md5FilePath := fmt.Sprintf("%s/md5.txt", localDir)
	err = generateMD5File(localDir, md5FilePath)
	if err != nil {
		log.Fatalf("生成MD5文件失败: %v", err)
	}
	log.Printf("MD5文件已生成: %s", md5FilePath)

	// 压缩目录到zip
	zipFilePath := fmt.Sprintf("./Save/%s.zip", now)
	err = zipDirectory(localDir, zipFilePath)
	if err != nil {
		log.Fatalf("压缩目录失败: %v", err)
	}
	log.Printf("目录已压缩到: %s", zipFilePath)

	// 压缩完成后删除原目录
	err = removeDirectory(localDir)
	if err != nil {
		log.Printf("删除原目录失败: %v", err)
	} else {
		log.Printf("原目录已删除: %s", localDir)
	}
}
