## 排程系統
- 註冊任務定時執行
- 支援 http 和 nsq
- [API Usage](./docs/swagger.yaml)

## Contents
- [cronJob 時區](#crontab-時區)
- [swag 安裝](#swag-安裝)

### crontab 時區

#### 時區
  - Asia/Taipei

#### 表達式說明
|秒|分|時|日|月|星期|
|:--:|:--:|:--:|:--:|:--:|:--:|
| * | * | * | * | * | * |
|0-59|0-59|0-23|1-31|1-12|0-7|

#### 範例
|表達式|說明|
|:--|:--|
| * * * * * * | 每秒 執行一次 |
| 0 */1 * * * * | 每分鐘 執行一次 |
| 0 0 */1 * * * | 每小時 執行一次 |
| 0 0 0 */1 * * | 每天00:00 執行一次 |
| 0 30 23 */1 * * | 每天23:30 執行一次 |
| 0 0 0 1 */1 * | 每月的第一天執行 |
| 0 30 21 * * 1 | 每周一21:30執行 |

|表達式|說明|等式|
|:--|:--|:--|
| @yearly (or @annually) | 每年1月1日 00:00:00 執行一次 | 0 0 0 1 1 * |
| @monthly | 每個月第一天的 00:00:00 執行一次  | 0 0 0 1 * * |
| @weekly | 每周周六的 00:00:00 執行一次 | 0 0 0 * * 0 |
| @daily (or @midnight) | 每天 00:00:00 執行一次 | 0 0 0 * * * |
| @hourly | 每小時執行一次 | 0 0 * * * * |
| @every duration | 指定時間間隔執行一次，如 @every 5s，每隔5秒執行一次。 |  0/5 * * * * * |

### swag 安裝

1. 下载swag：
```sh
go install github.com/swaggo/swag/cmd/swag@v1.16.2
```
```sh
go get github.com/swaggo/swag@v1.16.2
```
2. 產生docs文件
```sh
swag init

swag init --parseVendor

swag init --parseVendor --exclude vendor/github.com
```
- `--parseVendor` 允許解析vendor目錄
- `--exclude` 排除不解析目錄
