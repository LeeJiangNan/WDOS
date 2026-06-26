# WDOS Callback Buffer — SCF 云函数

接收 CRIP 回调 → 写入 COS → 昭阳拉取

## 部署步骤

### 1. 在腾讯云创建 COS 存储桶

```
控制台 → 对象存储 COS → 创建存储桶
- 名称：wdos-callback
- 地域：广州（ap-guangzhou）
- 访问权限：私有读写
```

记下存储桶全名，格式：`wdos-callback-1234567890`

### 2. 创建 SCF 云函数

```
控制台 → 云函数 SCF → 新建
- 函数类型：HTTP 触发函数
- 运行环境：Go 1.x
- 函数名称：wdos-callback-buffer
```

### 3. 编译上传

```bash
cd scf/
GOOS=linux GOARCH=amd64 go build -o main main.go
zip scf.zip main
# 在 SCF 控制台上传 scf.zip
```

### 4. 配置环境变量

在 SCF 控制台 → 环境变量：

| 变量 | 值 |
|------|-----|
| COS_BUCKET | wdos-callback-1234567890 |
| COS_REGION | ap-guangzhou |
| COS_SECRET_ID | AKIDxxxxxxxx |
| COS_SECRET_KEY | xxxxxxxx |

### 5. 获取触发器 URL

函数部署后，HTTP 触发器会生成 URL：
```
https://service-xxxxx-xxxxx.gz.apigw.tencentcs.com/release/
```

**这个 URL 就是 CRIP 的 callback 推送地址。**

### 6. 验证

```bash
# 健康检查
curl https://service-xxxxx-xxxxx.gz.apigw.tencentcs.com/release/

# 模拟 CRIP 推送
curl -X POST https://service-xxxxx-xxxxx.gz.apigw.tencentcs.com/release/ \
  -H "Content-Type: application/json" \
  -d @test_callback.json
```
