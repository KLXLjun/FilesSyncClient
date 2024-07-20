package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
)

var reader = bufio.NewReader(os.Stdin)

const SyncVersion = 0

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		NoColors:        true,
		TimestampFormat: "2006-01-02 15:04:05",
		FieldsOrder:     []string{"component", "category"},
	})

	logrus.Info("FilesSyncClient")
	logrus.Info(fmt.Sprintf("服务端同步格式版本: %d", SyncVersion))

	ex, err := os.Executable()
	if err != nil {
		logrus.Error("获取执行目录错误", err)
	}
	exPath := filepath.Dir(ex)
	logrus.Info("执行目录:", exPath)

	logrus.Trace("读入配置文件")
	if ok, loaderr := LoadConf(exPath); !ok {
		logrus.Error("解析配置文件时出错", loaderr)
		logrus.Error("回车键 (Enter) 结束进程")
		_, _ = reader.ReadString('\n')
		os.Exit(0)
	}

	if conf.ClientConf.WhiteList != nil {
		logrus.Info("检测到白名单文件列表:")
		for _, v := range conf.ClientConf.WhiteList {
			logrus.Info(v)
		}
		logrus.Info("")
	} else {
		logrus.Info("未检测到白名单文件列表")
	}

	localPath := path.Join(exPath, conf.ClientConf.Root)

	u, urlParseError := url.Parse(conf.ServerConf.Url)
	if urlParseError != nil {
		logrus.Error("解析服务器地址时出错", urlParseError)
		logrus.Error("回车键 (Enter) 结束进程")
		_, _ = reader.ReadString('\n')
		os.Exit(0)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		logrus.Error("服务器地址格式不正确 应为http或https 结果为", u.Scheme)
		logrus.Error("回车键 (Enter) 结束进程")
		_, _ = reader.ReadString('\n')
		os.Exit(0)
	}

	if u.Scheme == "http" {
		logrus.Warn("警告: 服务器地址为http")
	}

	listOk, listBytes := HttpRequest(fmt.Sprintf("%s/list/%s", conf.ServerConf.Url, conf.ServerConf.Check))
	if !listOk {
		logrus.Error("请检查配置是否正确!")
		logrus.Error("回车键 (Enter) 结束进程")
		_, _ = reader.ReadString('\n')
		os.Exit(0)
	}

	var list ListResult
	listJsonDecError := json.Unmarshal(listBytes, &list)
	if listJsonDecError != nil {
		logrus.Error("服务器列出列表的结果不正确")
		logrus.Error("回车键 (Enter) 结束进程")
		_, _ = reader.ReadString('\n')
		os.Exit(0)
	}

	if list.Ver != SyncVersion {
		logrus.Warn(fmt.Sprintf("警告: 服务端与客户端同步格式版本不一致! 客户端版本:%d 服务端版本:%d", SyncVersion, list.Ver))
		logrus.Error("按下回车键 (Enter) 确认并继续")
		_, _ = reader.ReadString('\n')
	}

	for _, indexPath := range list.Folder {
		encodePath := Base58Encode([]byte(indexPath))
		logrus.Info(fmt.Sprintf("开始项目 %s (%s) 的更新检查", indexPath, encodePath))
		nowScanPath := path.Join(localPath, indexPath)

		locallist := scan(nowScanPath)
		logrus.Info("计算完成")
		time.Sleep(20 * time.Millisecond)
		logrus.Info("正在与更新服务器进行通信")
		getok, resp := HttpRequest(fmt.Sprintf("%s/update/%s/%s", conf.ServerConf.Url, conf.ServerConf.Check, encodePath))
		if !getok {
			logrus.Error("很抱歉出现了错误,请按回车键 (Enter) 结束进程然后重新启动程序再试")
			_, _ = reader.ReadString('\n')
			os.Exit(0)
		}
		logrus.Trace(string(resp))

		bk := RJson{}
		jsonDecError := json.Unmarshal(resp, &bk)
		if err != nil {
			logrus.Error("服务器返回解析错误", jsonDecError)
			logrus.Error("很抱歉出现了错误,请按回车键 (Enter) 结束进程然后重新启动程序再试")
			_, _ = reader.ReadString('\n')
			os.Exit(0)
		}

		procsslist := make([]Process, 0)

		for _, local := range locallist {
			isRemove := true
			for _, remote := range bk.File {
				if local.FileName == remote.FileName && local.Hash == remote.Hash {
					isRemove = false
					continue
				}
				if local.FileName == remote.FileName && local.Hash != remote.Hash {
					continue
				}
			}
			if conf.ClientConf.WhiteList != nil {
				isWhiteListRow := false
				for _, v := range conf.ClientConf.WhiteList {
					if local.FileName == v {
						isWhiteListRow = true
						break
					}
				}
				if isWhiteListRow {
					continue
				}
			}
			if isRemove {
				procsslist = append(procsslist, Process{
					FilePath: local.FilePath,
					Status:   Delete,
				})
			}
		}

		for _, remote := range bk.File {
			isDownload := true
			for _, local := range locallist {
				if local.FileName == remote.FileName && local.Hash == remote.Hash {
					isDownload = false
					continue
				}
			}
			if isDownload {
				procsslist = append(procsslist, Process{
					FilePath:    path.Join(nowScanPath, remote.FileName),
					DownloadUrl: fmt.Sprintf("%s/dl/%s/%s/%s", conf.ServerConf.Url, conf.ServerConf.Check, encodePath, remote.Hash),
					Status:      Download,
				})
			}
		}

		logrus.Info("差异列表已就绪")

		for i, i2 := range procsslist {
			logrus.Debug(fmt.Sprintf("%d - %s [%s]", i, stat[i2.Status], path.Base(i2.FilePath)))
		}

		if len(procsslist) != 0 {
			logrus.Info(fmt.Sprintf("项目 %s 发现更新,操作数量为:%d", indexPath, len(procsslist)))
			processfunc(procsslist)
		} else {
			logrus.Info(fmt.Sprintf("项目 %s 没有更新", indexPath))
		}
	}

	logrus.Info("所有更新过程已结束,您可以按下回车键 (Enter) 结束进程或者关闭该窗口了! (倒计时15秒后自动关闭)")

	go func() {
		time.Sleep(15 * time.Second)
		os.Exit(0)
	}()

	reader.ReadString('\n')
	os.Exit(0)
}

func scan(mods string) []FileInfo {
	_ = os.MkdirAll(mods, 0755)
	templateFolder, rederr := os.Open(mods)
	if rederr != nil {
		logrus.Error("目录扫描失败,原因是:", rederr)
		logrus.Error("回车键 (Enter) 结束进程")
		reader.ReadString('\n')
		os.Exit(0)
	}
	filelist := make([]string, 0)
	templateInfos, _ := templateFolder.Readdir(-1)
	for _, info := range templateInfos {
		if !info.IsDir() {
			filelist = append(filelist, info.Name())
		}
	}

	local := make([]FileInfo, 0)
	bar := progressbar.Default(int64(len(filelist)), "正在计算文件哈希值")
	for _, s := range filelist {
		file, err := os.Open(path.Join(mods, s))
		if err != nil {
			logrus.Error("打开文件失败,原因是:", err)
			bar.Finish()
			logrus.Error("回车键 (Enter) 结束进程")
			reader.ReadString('\n')
			os.Exit(0)
		}
		resultHash := Sha3SumFile(file)
		file.Close()
		if path.Ext(s) == ".disabled" {
			local = append(local, FileInfo{
				FileName:  path.Base(path.Join(mods, s[:len(s)-9])),
				FilePath:  path.Join(mods, s),
				IsDisable: true,
				Hash:      resultHash,
			})
		} else {
			local = append(local, FileInfo{
				FileName:  path.Base(path.Join(mods, s)),
				FilePath:  path.Join(mods, s),
				IsDisable: false,
				Hash:      resultHash,
			})
		}
		bar.Add(1)
		time.Sleep(40 * time.Millisecond)
	}
	bar.Finish()
	return local
}

func processfunc(process []Process) {
	for i, p := range process {
		if p.Status == Delete {
			os.Remove(p.FilePath)
			logrus.Info(fmt.Sprintf("[%d/%d] 删除 %s", i+1, len(process), p.FilePath))
		} else if p.Status == Download {
			nocount := 6  //错误重试计数
			sleepout := 4 //重试等待时间

			for {
				req, _ := http.NewRequest("GET", p.DownloadUrl, nil)
				resp, resperr := http.DefaultClient.Do(req)
				if resperr != nil {
					logrus.Warn("请求时发生了错误", resperr)
					resp.Body.Close()
					nocount = nocount - 1
					if nocount == 0 {
						break
					}
					sleeptime := time.Second * time.Duration(sleepout)
					logrus.Trace(fmt.Sprintf("[%d-%d] 下载失败, %.0fs后重试,剩余重试次数 %d 次", i+1, len(process), sleeptime.Seconds(), nocount))
					time.Sleep(sleeptime)
					sleepout += sleepout
				} else {
					mkdirallerr := os.MkdirAll(path.Dir(p.FilePath), 0777)
					if mkdirallerr != nil {
						logrus.Warn("创建文件路径文件夹时出错", path.Dir(p.FilePath), mkdirallerr)
					}

					f, _ := os.OpenFile(p.FilePath, os.O_CREATE|os.O_WRONLY, 0777)
					bar := progressbar.DefaultBytes(
						resp.ContentLength,
						fmt.Sprintf("[%d/%d] 下载", i+1, len(process)),
					)

					io.Copy(io.MultiWriter(f, bar), resp.Body)
					resp.Body.Close()
					f.Close()
					break
				}
			}
			if nocount == 0 {
				logrus.Warn(fmt.Sprintf("[%d-%d] 下载失败 链接:%s", i+1, len(process), p.DownloadUrl))
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
}

var stat = []string{0: "下载", 1: "删除"}

const (
	Download = iota
	Delete
)
