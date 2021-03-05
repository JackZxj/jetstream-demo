# jetstream demo

+ nats cluster 安装详见 [ec-co.md](ec-co.md)
+ 单向网络测试详见 [One-way_network_test.md](One-way_network_test.md)

## 效果

1. edge 端每隔 5~14 秒 (间隔随机) 发送一条消息
2. cloud 端每隔 5 秒主动拉取一次消息，超时时间为5秒

## 使用方式

``` bash
$ docker build -t 172.31.0.7:5000/js-demo:v0.5 .
$ docker push 172.31.0.7:5000/js-demo:v0.5
$ kubectl apply -f deployment.yaml
$ kubectl delete -f deployment.yaml
```

## 效果展示

**edge 侧：**

``` BASH
$ kubectl logs js-demo-edge-545c7774d9-8wzzn -f
2021/02/25 01:49:24 main.go:40: I get some environment variables: | server: nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 | role: edge | subject: mysqldb2.1 | stream: mysqldb2 | consumer: mysqldb2 |
2021/02/25 01:49:24 main.go:60: Run with those environment variables: | URL: nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 | ROLE: edge | SUBJECT: mysqldb2.1 | STREAM: mysqldb2 | CONSUMER: mysqldb2 |
2021/02/25 01:49:24 main.go:77: MSG: {"stream":"mysqldb2","seq":433} 	 Sleep time: 9s 
2021/02/25 01:49:33 main.go:77: MSG: {"stream":"mysqldb2","seq":434} 	 Sleep time: 8s 
2021/02/25 01:49:41 main.go:77: MSG: {"stream":"mysqldb2","seq":435} 	 Sleep time: 7s 
2021/02/25 01:49:48 main.go:77: MSG: {"stream":"mysqldb2","seq":436} 	 Sleep time: 10s 
2021/02/25 01:49:58 main.go:77: MSG: {"stream":"mysqldb2","seq":437} 	 Sleep time: 9s 
2021/02/25 01:50:07 main.go:77: MSG: {"stream":"mysqldb2","seq":438} 	 Sleep time: 9s 
2021/02/25 01:50:16 main.go:77: MSG: {"stream":"mysqldb2","seq":439} 	 Sleep time: 9s 
2021/02/25 01:50:25 main.go:77: MSG: {"stream":"mysqldb2","seq":440} 	 Sleep time: 13s 
2021/02/25 01:50:38 main.go:77: MSG: {"stream":"mysqldb2","seq":441} 	 Sleep time: 7s
···
```

**cloud 侧：**
``` BASH
$ kubectl logs js-demo-cloud-f7684ccb-qx5zx -f
2021/02/25 01:49:25 main.go:40: I get some environment variables: | server: nats://ce1:ce1@nats-cloud-0.nats-cloud.default.svc.cluster.local:4222 | role: cloud | subject: mysqldb2.1 | stream: mysqldb2 | consumer: mysqldb2 |
2021/02/25 01:49:25 main.go:60: Run with those environment variables: | URL: nats://ce1:ce1@nats-cloud-0.nats-cloud.default.svc.cluster.local:4222 | ROLE: cloud | SUBJECT: mysqldb2.1 | STREAM: mysqldb2 | CONSUMER: mysqldb2 |
[01:49:25] subj: mysqldb2.1 / tries: 1 / cons seq: 953 / str seq: 433 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:49:24]
Acknowledged message: 

[01:49:33] subj: mysqldb2.1 / tries: 1 / cons seq: 954 / str seq: 434 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:49:33]
Acknowledged message: 

[01:49:41] subj: mysqldb2.1 / tries: 1 / cons seq: 955 / str seq: 435 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:49:41]
Acknowledged message: 

[01:49:48] subj: mysqldb2.1 / tries: 1 / cons seq: 956 / str seq: 436 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:49:48]
Acknowledged message: 

2021/02/25 01:49:58 main.go:112: no message received
[01:50:03] subj: mysqldb2.1 / tries: 2 / cons seq: 958 / str seq: 437 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:49:58]
Acknowledged message: 

[01:50:08] subj: mysqldb2.1 / tries: 1 / cons seq: 959 / str seq: 438 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:50:07]
Acknowledged message: 

[01:50:16] subj: mysqldb2.1 / tries: 1 / cons seq: 960 / str seq: 439 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:50:16]
Acknowledged message: 

[01:50:25] subj: mysqldb2.1 / tries: 1 / cons seq: 961 / str seq: 440 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:50:25]
Acknowledged message: 

2021/02/25 01:50:35 main.go:112: no message received
[01:50:40] subj: mysqldb2.1 / tries: 1 / cons seq: 962 / str seq: 441 / pending: 0
Data:  Hello, can you hear me?	[Edge time: 2021-02-25 01:50:38]
Acknowledged message: 

···
```
