# FilesSyncClient

此为 FilesSync (文件同步) 的客户端，[服务端在此](https://github.com/KLXLjun/FilesSyncServer)

写的很简单，也不是很好，主要是为了实现同步功能，有bug或是功能意见请开一个新的issue

# 使用说明

启动程序后，会生成一个名为 ```config.yaml``` 的配置文件，内容如下：

```yaml
client:
    root: ./
    whitelist:
        - example.jar
server:
    url: http://example.com
    check: example
```

## 配置说明

### 客户端配置 (`client`)

- `root`：同步的根目录。
- `whitelist`：白名单文件名列表，列出不需要更新的文件名。

### 服务端配置 (`server`)

- `url`：服务器的请求地址，需包含 `http` 或 `https`。
- `check`：服务端配置的查询码（check）。
