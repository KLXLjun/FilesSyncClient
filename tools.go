package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/akamensky/base58"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"
)

func FileisExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}

func FileSize(path string) int64 {
	i, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return math.MinInt64
		}
		return math.MinInt64
	}
	return i.Size()
}

func HttpRequest(webUrl string) (bool, []byte) {
	request, _ := http.NewRequest("GET", webUrl, nil)
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36")
	request.Header.Set("connection", "close")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 30, //超时时间
	}

	resp, err := client.Do(request)
	if err != nil && err != io.EOF {
		logrus.Warn("请求时发生了错误 ", err)
		return false, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		logrus.Warn(fmt.Sprintf("请求时服务器抛出错误 %d %s", resp.StatusCode, webUrl))
		return false, nil
	}

	body, ReadAllError := io.ReadAll(resp.Body)
	if ReadAllError != nil {
		logrus.Warn("尝试读取全部数据时出现错误 ", ReadAllError)
		return false, nil
	}
	return true, body
}

func Sha3SumFile(file io.Reader) string {
	hash := sha3.New224()
	_, err := io.Copy(hash, file)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func ReadLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	r := bufio.NewReader(f)
	for {
		bytes, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return lines, err
		}
		lines = append(lines, string(bytes))
	}
	return lines, nil
}

func ReadOrCreateFile(filename string, content []byte) ([]byte, error) {
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return []byte{}, fmt.Errorf("不能读取文件: %v", err)
		}
		return data, nil
	} else if os.IsNotExist(err) {
		err = os.WriteFile(filename, []byte(content), 0755)
		if err != nil {
			return []byte{}, fmt.Errorf("不能创建文件: %v", err)
		}
		return content, nil
	} else {
		return []byte{}, fmt.Errorf("检查文件出错: %v", err)
	}
}

func Base58Encode(input []byte) string {
	return base58.Encode(input)
}

func Base58Decode(input string) (string, error) {
	decoded, err := base58.Decode(input)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	return string(decoded), nil
}
