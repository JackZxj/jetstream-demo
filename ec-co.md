# nats 云边协同

## 一、 NATS-Server 集群部署

仅用于测试用部署：

``` BASH
# 没有准备 pv 用于 JetStream 存储时用 no-pv 版本
$ kubectl apply -f nats-cloud-no-pv.yaml
# 修改部署的节点
$ kubectl apply -f nats-edge-no-pv.yaml
```

**注：**
以下操作均在测试环境中部署。实际部署时，将以下部署与测试中的相关链接替换为实际部署的地址。

## 二、 Operator 部署与使用

``` BASH
# 部署 crds
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nack/main/deploy/crds.yml
# 设置 rbac
$ kubectl apply -f https://raw.githubusercontent.com/nats-io/nack/main/deploy/rbac.yml
# 获取 nack controller
$ wget https://raw.githubusercontent.com/nats-io/nack/main/deploy/deployment.yml
$ vi deployment.yml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jetstream-controller
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      name: jetstream-controller
  template:
    metadata:
      labels:
        name: jetstream-controller
    spec:
      serviceAccountName: jetstream-controller
      containers:
      - name: jsc
        image: connecteverything/jetstream-controller:0.1.0
        imagePullPolicy: IfNotPresent
        command:
        - /jetstream-controller
        - -s=nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 # 修改此处连接的端点为 leafnode 端点

$ kubectl apply -f deployment.yml
# 注: 目前 nack 还不完善，不支持动态切换连接的用户以及节点，若要更换用户需要更新 deployment 后重启容器
```

jetstream operator支持创建 Stream/StreamTemplate 和 Consumer 三种类型的资源，资源完整定义如下：

``` yaml
---
apiVersion: jetstream.nats.io/v1beta1
kind: StreamTemplate
metadata:
  name: mystreamtemplate
spec:
  name: mystreamtemplate
  maxStreams: 2
  subjects: ["orders.*"]
  retention: limits
  maxConsumers: -1
  maxMsgs: -1
  maxBytes: -1
  maxAge: 1h
  maxMsgSize: -1
  storage: memory
  replicas: 1
  noAck: false
  discard: new
  duplicateWindow: 2s
---
apiVersion: jetstream.nats.io/v1beta1
kind: Stream
metadata:
  name: mystream
spec:
  name: mystream
  subjects: ["orders.*"]
  retention: limits
  maxConsumers: -1
  maxMsgs: -1
  maxBytes: -1
  maxAge: 1h
  maxMsgSize: -1
  storage: memory
  replicas: 1
  noAck: false
  discard: new
  duplicateWindow: 2s
---
apiVersion: jetstream.nats.io/v1beta1
kind: Consumer
metadata:
  name: my-consumer
spec:
  streamName: mystream
  deliverPolicy: last
#   optStartSeq: 0 # deliverPolicy=byStartSequence 时需要此项
#   optStartTime: "2021-01-02T15:00:00Z" # deliverPolicy=byStartTime 时需要此项
  durableName: my-consumer
  deliverSubject: my-consumer.orders
  ackPolicy: none
  ackWait: 1ns
  maxDeliver: -1
  filterSubject: my-consumer.order
  replayPolicy: instant
  sampleFreq: 100
```

## 三、 创建路由

### 使用示例

参数参考: [jetstream](https://github.com/nats-io/jetstream#creating)

``` BASH
# 通过 NATS cli 部署
$ nats str -s nats://s1:s1@localhost:4222 add mysqldb1 --subjects="mysqldb1.>" --storage=file --retention=workq --discard=new --max-msgs=-1 --max-bytes=-1 --max-age=-1 --max-msg-size=-1 --dupe-window=2s --replicas=1

$ nats -s nats://s1:s1@localhost:4222 con add mysqldb1 mysqldb1 [--target="mtest.1"] --deliver=all --replay=instant --filter="mysqldb1.1" --max-deliver=-1 --max-pending=1 --pull

# 下面有两种方式部署到 k8s
################################################################
#                                                              #
#                 1. Deploy by operator yaml                   #
#                                                              #
################################################################
# stream
$ vi stream-mysqldb2.yaml
apiVersion: jetstream.nats.io/v1beta1
kind: Stream
metadata:
  name: mysqldb2
spec:
  name: mysqldb2
  subjects: ["mysqldb2.>"]
  retention: workqueue
  maxConsumers: -1
  maxMsgs: -1
  maxBytes: -1
  maxAge: 1h # -1有问题
  maxMsgSize: -1
  storage: file
  replicas: 1
  noAck: false
  discard: new
  duplicateWindow: 2s

$ kubectl apply -f stream-mysqldb2.yaml
# $ kubectl delete -f stream-mysqldb2.yaml

# consumer
$ vi consumer-mysqldb2.yaml
apiVersion: jetstream.nats.io/v1beta1
kind: Consumer
metadata:
  name: mysqldb2
spec:
  streamName: mysqldb2
  deliverPolicy: all
  replayPolicy: instant
  filterSubject: mysqldb2.1
  maxDeliver: -1
  durableName: mysqldb2
  ackPolicy: explicit # consumer in pull mode requires explicit ack policy

$ kubectl apply -f consumer-mysqldb2.yaml
# $ kubectl delete -f consumer-mysqldb2.yaml

$ kubectl get stream
NAME       STATE     STREAM NAME   SUBJECTS
mysqldb2   Created   mysqldb2      [mysqldb2.>]

$ kubectl get consumer
NAME       STATE     STREAM     CONSUMER   ACK POLICY
mysqldb2   Created   mysqldb2   mysqldb2   explicit

################################################################
#                                                              #
#                 2. Deploy by operator api                    #
#                                                              #
################################################################
# 窗口1 (临时开放 k8s apiserver 的 8001 端口，仅测试用，生产环境需要配置 token, 使用 token 访问 k8s api)
$ kubectl proxy
# 窗口2
########################
#        Stream        #
########################
# Create a new namespaced stream: POST /apis/{group}/{version}/namespaces/{namespace}/{plural}
curl -v -X POST http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/streams --header 'Content-Type: application/json' --header 'Accept: application/json' -d '{
    "apiVersion": "jetstream.nats.io/v1beta1",
    "kind": "Stream",
    "metadata": {
        "name": "mysqldb3"
    },
    "spec": {
        "discard": "new",
        "duplicateWindow": "2s",
        "maxAge": "1h",
        "maxBytes": -1,
        "maxConsumers": -1,
        "maxMsgSize": -1,
        "maxMsgs": -1,
        "name": "mysqldb3",
        "noAck": false,
        "replicas": 1,
        "retention": "workqueue",
        "storage": "file",
        "subjects": [
            "mysqldb3.\u003e"
        ]
    }
}'
# List namespaced streams: GET /apis/{group}/{version}/namespaces/{namespace}/{plural}
curl http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/streams
# Get a namespaced stream: GET /apis/{group}/{version}/namespaces/{namespace}/{plural}/{name}
curl http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/streams/mysqldb3
# Update a namespaced stream: PUT /apis/{group}/{version}/namespaces/{namespace}/{plural}/{name}
# 更新: "maxAge": "2h" (修改get请求获得的json)
curl -v -X PUT http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/streams/mysqldb3 --header 'Content-Type: application/json' --header 'Accept: application/json' -d '{"apiVersion":"jetstream.nats.io/v1beta1","kind":"Stream","metadata":{"creationTimestamp":"2021-02-23T02:41:13Z","finalizers":["streamfinalizer.jetstream.nats.io"],"generation":1,"managedFields":[{"apiVersion":"jetstream.nats.io/v1beta1","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{".":{},"f:discard":{},"f:duplicateWindow":{},"f:maxAge":{},"f:maxBytes":{},"f:maxConsumers":{},"f:maxMsgSize":{},"f:maxMsgs":{},"f:name":{},"f:noAck":{},"f:replicas":{},"f:retention":{},"f:storage":{},"f:subjects":{}}},"manager":"curl","operation":"Update","time":"2021-02-23T02:41:13Z"},{"apiVersion":"jetstream.nats.io/v1beta1","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:finalizers":{".":{},"v:\"streamfinalizer.jetstream.nats.io\"":{}}},"f:status":{".":{},"f:conditions":{},"f:observedGeneration":{}}},"manager":"jetstream-controller","operation":"Update","time":"2021-02-23T02:41:13Z"}],"name":"mysqldb3","namespace":"default","resourceVersion":"18939596","selfLink":"/apis/jetstream.nats.io/v1beta1/namespaces/default/streams/mysqldb3","uid":"94cca1bb-6341-4d89-a37b-fe83a2336884"},"spec":{"discard":"new","duplicateWindow":"2s","maxAge":"2h","maxBytes":-1,"maxConsumers":-1,"maxMsgSize":-1,"maxMsgs":-1,"name":"mysqldb3","noAck":false,"replicas":1,"retention":"workqueue","storage":"file","subjects":["mysqldb3.\u003e"]},"status":{"conditions":[{"lastTransitionTime":"2021-02-23T02:41:13.404254221Z","message":"Stream successfully created","reason":"Created","status":"True","type":"Ready"}],"observedGeneration":1}}'
# Delete a namespaced stream: DELETE /apis/{group}/{version}/namespaces/{namespace}/{plural}/{name}
curl -v -X DELETE http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/streams/mysqldb3

########################
#       Consumer       #
########################
# Create a new namespaced consumer: POST /apis/{group}/{version}/namespaces/{namespace}/{plural}
curl -v -X POST http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/consumers --header 'Content-Type: application/json' --header 'Accept: application/json' -d '{
    "apiVersion": "jetstream.nats.io/v1beta1",
    "kind": "Consumer",
    "metadata": {
        "name": "mysqldb3"
    },
    "spec": {
        "ackPolicy": "explicit",
        "ackWait": "1ns",
        "deliverPolicy": "all",
        "deliverSubject": "",
        "durableName": "mysqldb3",
        "filterSubject": "mysqldb2.2",
        "maxDeliver": -1,
        "optStartSeq": 0,
        "optStartTime": "",
        "rateLimitBps": 0,
        "replayPolicy": "instant",
        "sampleFreq": "",
        "streamName": "mysqldb2"
    }
}'
# List namespaced consumers: GET /apis/{group}/{version}/namespaces/{namespace}/{plural}
curl http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/consumers
# Get a namespaced consumer: GET /apis/{group}/{version}/namespaces/{namespace}/{plural}/{name}
curl http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/consumers/mysqldb3
# Update a namespaced consumer: PUT /apis/{group}/{version}/namespaces/{namespace}/{plural}/{name}
# 更新: "ackWait": "2ns" (修改get请求获得的json)
curl -v -X PUT http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/consumers/mysqldb3 --header 'Content-Type: application/json' --header 'Accept: application/json' -d '{"apiVersion":"jetstream.nats.io/v1beta1","kind":"Consumer","metadata":{"creationTimestamp":"2021-02-23T03:11:45Z","finalizers":["consumerfinalizer.jetstream.nats.io"],"generation":1,"managedFields":[{"apiVersion":"jetstream.nats.io/v1beta1","fieldsType":"FieldsV1","fieldsV1":{"f:spec":{".":{},"f:ackPolicy":{},"f:ackWait":{},"f:deliverPolicy":{},"f:deliverSubject":{},"f:durableName":{},"f:filterSubject":{},"f:maxDeliver":{},"f:optStartSeq":{},"f:optStartTime":{},"f:rateLimitBps":{},"f:replayPolicy":{},"f:sampleFreq":{},"f:streamName":{}}},"manager":"curl","operation":"Update","time":"2021-02-23T03:11:45Z"},{"apiVersion":"jetstream.nats.io/v1beta1","fieldsType":"FieldsV1","fieldsV1":{"f:metadata":{"f:finalizers":{".":{},"v:\"consumerfinalizer.jetstream.nats.io\"":{}}},"f:status":{".":{},"f:conditions":{},"f:observedGeneration":{}}},"manager":"jetstream-controller","operation":"Update","time":"2021-02-23T03:11:45Z"}],"name":"mysqldb3","namespace":"default","resourceVersion":"18948041","selfLink":"/apis/jetstream.nats.io/v1beta1/namespaces/default/consumers/mysqldb3","uid":"088107bd-9925-4f31-9af5-6cc50934d104"},"spec":{"ackPolicy":"explicit","ackWait":"2ns","deliverPolicy":"all","deliverSubject":"","durableName":"mysqldb3","filterSubject":"mysqldb2.2","maxDeliver":-1,"optStartSeq":0,"optStartTime":"","rateLimitBps":0,"replayPolicy":"instant","sampleFreq":"","streamName":"mysqldb2"},"status":{"conditions":[{"lastTransitionTime":"2021-02-23T03:11:45.580736439Z","message":"Consumer successfully created","reason":"Created","status":"True","type":"Ready"}],"observedGeneration":1}}'
# Delete a namespaced consumer: DELETE /apis/{group}/{version}/namespaces/{namespace}/{plural}/{name}
curl -v -X DELETE http://127.0.0.1:8001/apis/jetstream.nats.io/v1beta1/namespaces/default/consumers/mysqldb3
```

### 创建结果

``` BASH
# 进入一个带有 nats 应用的容器查看结果
$ kubectl exec -it nats-edge-0 -- sh

# 以下操作在容器中进行，需要注明使用的用户以及连接的地址
# 查看当前 jetstream 中的 Stream
$ nats -s nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 str ls
Streams:

	mysqldb2

# 查看某个 stream 的详情
$ nats -s nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 str info mysqldb2
Information for Stream mysqldb2 created 2021-02-22T10:25:44Z

Configuration:

             Subjects: mysqldb2.>
     Acknowledgements: true
            Retention: Memory - WorkQueue
             Replicas: 1
       Discard Policy: New
     Duplicate Window: 2s
     Maximum Messages: unlimited
        Maximum Bytes: unlimited
          Maximum Age: 1h0m0s
 Maximum Message Size: unlimited
    Maximum Consumers: unlimited

State:

             Messages: 0
                Bytes: 0 B
             FirstSeq: 0
              LastSeq: 0
     Active Consumers: 1

# 查看某个 stream 中的所有 consumer
$ nats -s nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 con ls mysqldb2
Consumers for Stream mysqldb2:

	mysqldb2

# 查看 consumer 的详情
$ nats -s nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 con info mysqldb2 mysqldb2
Information for Consumer mysqldb2 > mysqldb2 created 2021-02-22T10:31:34Z

Configuration:

        Durable Name: mysqldb2
           Pull Mode: true
      Filter Subject: mysqldb2.1
         Deliver All: true
          Ack Policy: Explicit
            Ack Wait: 1ns
       Replay Policy: Instant

State:

   Last Delivered Message: Consumer sequence: 0 Stream sequence: 0
     Acknowledgment floor: Consumer sequence: 0 Stream sequence: 0
         Outstanding Acks: 0
     Redelivered Messages: 0
    Waiting Pull Requests: 0
     Unprocessed Messages: 0
```

### 收发消息测试

#### NATS CLI 测试

+ 发送

  在边缘节点1处，使用用户s1进行登录，向subject mysqldb2.1中发送消息。

  ```bash
  # 进入一个带有nats cli的边缘节点
  $ kubectl exec -it nats-edge-0 -- sh
  # 发送消息
  $ nats -s nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 request mysqldb2.1 hello1
  07:34:33 Sending request on "mysqldb2.1"
  07:34:33 Received on "_INBOX.ESqSDUbJ89B01rNYV4OhUn.rYEdVWRW" rtt 887.311µs
  {"stream":"mysqldb2","seq":1}

  $ nats -s nats://s1:s1@nats-edge-0.nats-edge.default.svc.cluster.local:4222 request mysqldb2.1 hello2
  07:39:09 Sending request on "mysqldb2.1"
  07:39:09 Received on "_INBOX.VnBjUqZdvlYnVzucbdZUr2.xt4CmdTI" rtt 565.006µs
  {"stream":"mysqldb2","seq":2}
  ```

+ 接收

  在云服务器处，使用用户ce1进行登录，使用stream mysqldb2 consumer mysqldb2 进行接收

  ``` bash
  # 进入一个带有nats cli的云端容器
  $ kubectl exec -it nats-cloud-0 -- sh
  # 接收消息
  $ nats -s nats://ce1:ce1@nats-cloud-0.nats-cloud.default.svc.cluster.local:4222 con next mysqldb2 mysqldb2
  [07:38:43] subj: mysqldb2.1 / tries: 1 / cons seq: 1 / str seq: 1 / pending: 0

  hello1

  Acknowledged message

  $ nats -s nats://ce1:ce1@nats-cloud-0.nats-cloud.default.svc.cluster.local:4222 req '$JS.API.CONSUMER.MSG.NEXT.mysqldb2.mysqldb2' '1'
  07:39:42 Sending request on "$JS.API.CONSUMER.MSG.NEXT.mysqldb2.mysqldb2"
  07:39:42 Received on "mysqldb2.1" rtt 880.71µs
  hello2

  ```

#### NATS package 测试

NATS go package 使用方式以及部署测试，请参考 demo 文件夹中的 [源码](main.go) 以及 [说明文档](README.md)。