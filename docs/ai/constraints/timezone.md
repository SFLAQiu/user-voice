# 时区处理规范

不遵守本规范会导致时间数据偏移 8 小时，属于硬约束。

## MySQL 连接配置

DSN 必须配置 `parseTime=True&loc=Local`，确保 MySQL driver 将 Go `time.Time` 按本地时区（中国北京时间 UTC+8）解释并转换。

## 前端传 UTC，后端转本地

- 前端 `dayjs.toISOString()` 输出带 `Z` 后缀的 UTC 时间（如 `2026-06-08T22:04:00.000Z`）
- 后端 `ParseAbsoluteTime` 解析 RFC3339 后必须调用 `t.Local()` 转为本地时区时刻
- 若不转换，MySQL driver 会把 UTC 时间当作本地时间写入查询（UTC 22:04 被错误理解为北京 22:04，而非正确的北京 06:04）

## 无时区后缀视为本地时区

无 `Z` 后缀的 ISO8601 和 MySQL DATETIME 格式字符串使用 `time.ParseInLocation` 解析，视为本地时区。

## 时间截断使用本地日期边界

- `Truncate(24h)` 对齐到 UTC 天边界而非本地天边界，在 +8 时区会偏移 8 小时
- 1d/1w 粒度的截断必须使用 `time.Date()` / `ISOWeek()` 按本地日期截断
- 1m/1h 粒度可用 `Truncate`

## 仪表盘趋势图零值填充

- `ZeroFill` 在 SQL 查询后补齐缺失时间桶（`value=0`）
- 最小填充步长 15s（仅对 1s bucket 生效），其他粒度按自然步长填充
- 填充的 x 值时间格式必须与 MySQL `DATE_FORMAT` 输出一致（本地时区）
