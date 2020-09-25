## 运行准备

1. 创建机器人, 并且用机器人创建 2 个 mixin network 的用户，然后替换 ./configs/yml.go.example 相关的信息
2. 安装 postgresql, 导入 ./models/schema.sql 

## 运行程序

go build && ./multisig

## 多签测试

给上面机器人发送任意的文字消息，机器会返回 CNB 支付按钮, 支付完成即可收到返还的多签转帐。
