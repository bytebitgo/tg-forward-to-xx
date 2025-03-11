# 数据迁移工具使用说明

## 功能说明

此工具用于修复以下两个问题：
1. 历史消息中缺失的群组名称字段
2. 无法正确解析的表情符号和特殊字符

工具会遍历所有历史消息记录，进行如下处理：
- 为空的 `GroupName` 字段添加群组名称
- 将无法解析的表情符号替换为 "Emoji 解析失败"

## 编译步骤

1. 进入项目目录：
```bash
cd tg-forward-to-xx
```

2. 编译迁移工具：
```bash
go build -o migrate cmd/migrate/main.go
```

## 使用方法

### 1. 准备工作

1. 停止主程序：
```bash
systemctl stop tg-forward
```

2. 备份数据库文件：
```bash
cp -r /var/lib/tg-forward/chat_history /var/lib/tg-forward/chat_history_backup
```

### 2. 执行迁移

1. 首先使用预览模式，查看需要更新的记录数量：
```bash
./migrate -config /etc/tg-forward/config.yaml -dry-run
```

2. 如果预览结果正确，执行实际迁移：
```bash
./migrate -config /etc/tg-forward/config.yaml
```

### 3. 验证结果

1. 导出部分记录检查：
```bash
curl -o check.csv "http://localhost:8080/api/chat/history/export?chat_id=YOUR_CHAT_ID&start_time=2024-03-01T00:00:00Z&end_time=2024-03-11T23:59:59Z"
```

2. 使用文本编辑器或 Excel 打开 check.csv，确认群组名称字段已正确填充

### 4. 完成迁移

1. 重启主程序：
```bash
systemctl start tg-forward
```

2. 检查日志确认程序正常运行：
```bash
journalctl -u tg-forward -f
```

## 命令行参数说明

- `-config`：指定配置文件路径
  - 默认值：`/etc/tg-forward/config.yaml`
  - 示例：`./migrate -config /path/to/config.yaml`

- `-dry-run`：预览模式，不实际修改数据
  - 默认值：`false`
  - 示例：`./migrate -dry-run`

## 迁移过程说明

1. 工具会读取配置文件中的群组 ID 列表
2. 遍历所有历史消息记录
3. 对每条记录进行以下处理：
   - 如果 GroupName 为空，设置为 "群组(群组ID)" 格式
   - 如果消息内容包含无法解析的字符，替换为 "Emoji 解析失败"
   - 将需要更新的记录添加到批处理队列
4. 每处理 1000 条记录提交一次批处理
5. 完成后显示处理统计信息

## 注意事项

1. 数据安全：
   - 执行迁移前必须备份数据库
   - 迁移时确保主程序已停止运行
   - 建议先使用 `-dry-run` 预览变更

2. 性能考虑：
   - 迁移过程使用批处理方式，每1000条记录提交一次
   - 大量数据迁移可能需要较长时间
   - 建议在系统负载较低时执行

3. 错误处理：
   - 迁移过程中的错误会记录到日志
   - 如果迁移中断，可以重新执行
   - 已更新的记录不会重复更新

4. 回滚方案：
   - 如果迁移结果不理想，可以使用备份恢复：
   ```bash
   systemctl stop tg-forward
   rm -rf /var/lib/tg-forward/chat_history
   cp -r /var/lib/tg-forward/chat_history_backup /var/lib/tg-forward/chat_history
   systemctl start tg-forward
   ```

## 常见问题

1. 配置文件不存在：
   - 确认配置文件路径是否正确
   - 检查配置文件权限

2. 数据库访问错误：
   - 确认数据库路径配置正确
   - 检查目录权限
   - 确保没有其他程序正在访问数据库

3. 内存使用过高：
   - 工具使用批处理方式，内存使用可控
   - 如果数据量特别大，建议在内存充足时执行

4. 迁移中断：
   - 可以直接重新执行迁移命令
   - 已更新的记录不会重复更新 