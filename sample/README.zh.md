分布式多签示例程序, 线上测试机器人 id: 7000101488

## 线上机器人使用方法

回复任意消息，并支付 CNB 到多签地址，机器人会自动签名并返回 

## 运行准备

该示例是 2/3 签名，所以运行该示例需要有 3 个 mixin network 用户 (或机器人), 至少一个为机器人用户, 用于跟 Mixin Messenger 用户交互。

1. 安装 postgresql, 并导入相关数据结构 ./models/schema.sql 
2. 到 [developers dashboard](https://developers.mixin.one/dashboard) 申请机器人，以及相关私钥等信息
3. 如果其它 2 个用户同样是机器人，同第 2 步, 如果是 mixin network 用户，需要用两个 api, `post /users` 及 `post /pin/update` 来创建 network 用户跟 pin
4. 用 2, 3 拿到的用户信息，`cp yml.go.example yml.go` 并更新 mixin 下相关信息, 分别启动, 注意目前只能有一个 master 跟用户交互

## 多签测试

给相关机器人发送任意的消息，机器会返回多签的支付链接, 支付完成会收到返还的多签转帐。
