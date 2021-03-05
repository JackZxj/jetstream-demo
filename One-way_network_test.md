# 单边网络测试

## 环境

``` BASH
$ kubectl get no -o wide
NAME              STATUS   ROLES         AGE    VERSION                   INTERNAL-IP       EXTERNAL-IP   OS-IMAGE                KERNEL-VERSION           CONTAINER-RUNTIME
10.110.26.178     Ready    master,node   114d   v1.18.3                   10.110.26.178     <none>        CentOS Linux 7 (Core)   3.10.0-957.el7.x86_64    docker://19.3.0
centos78-0        Ready    node          114d   v1.18.3                   192.168.122.148   <none>        CentOS Linux 7 (Core)   3.10.0-1127.el7.x86_64   docker://19.3.0
centos78-1        Ready    node          114d   v1.18.3                   192.168.122.242   <none>        CentOS Linux 7 (Core)   3.10.0-1127.el7.x86_64   docker://19.3.0
centos78-edge-0   Ready    agent,edge    123m   v1.19.3-kubeedge-v1.5.0   192.168.122.116   <none>        CentOS Linux 7 (Core)   3.10.0-1127.el7.x86_64   docker://19.3.0
```

在 edge 上使用 iptable 禁止 matser 访问: `iptables -I INPUT -s 192.168.122.1 -j DROP` (注: edge以及所有worker节点都是master主机上的 kvm 虚机，所有虚机都通过虚拟网卡 192.168.122.1 与外网连通)

## 配置

``` BASH
$ kubectl apply -f nats-cloud-no-pv.yaml
$ kubectl get po -owide | grep nats-cloud
nats-cloud-0                            3/3     Running   0          24h   10.253.0.83       10.110.26.178     <none>           <none>
nats-cloud-1                            3/3     Running   3          9d    10.253.2.218      centos78-0        <none>           <none>
nats-cloud-2                            3/3     Running   3          9d    10.253.1.169      centos78-1        <none>           <none>

# 修改 nats-edge-no-pv-offline.yaml 中 comfigmap 中的 remotes 地址为上面获取的pod的ip，然后再部署
$ kubectl apply -f nats-edge-no-pv-offline.yaml
$ kubectl get po -owide | grep nats-edge
nats-edge-0                             1/1     Running   0          63m   192.168.122.116   centos78-edge-0   <none>           <none>
```

完成上述云边部署后，部署 nack operator，controller 的 Deployment 要连接到云端 nats-cloud 集群

## 验证

``` BASH
# 在云端
$ kubectl exec -it nats-cloud-0 -- sh
Defaulting container name to nats.
Use 'kubectl describe pod/nats-cloud-0 -n default' to see all of the containers in this pod.
$ nats -s nats://ce1:ce1@127.0.0.1:4222 str ls
No Streams defined

# 在边缘端
$ docker exec -it 7a2b5f14ee07 sh
$ nats -s nats://s1:s1@127.0.0.1:4222 str ls
No Streams defined
$ nats -s nats://s1:s1@127.0.0.1:4222 account info
Connection Information:

               Client ID: 9
               Client IP: 127.0.0.1
                     RTT: 383.433µs
       Headers Supported: true
         Maximum Payload: 1.0 MiB
           Connected URL: nats://s1:s1@127.0.0.1:4222
       Connected Address: 127.0.0.1:4222
     Connected Server ID: NBXWX2LFUTID2ISCYATOFNNVEDDRCYQVK2S6LVCOIJLXMFX4KQKTT62O
   Connected Server Name: edge1

JetStream Account Information:

           Memory: 0 B of 16 EiB
          Storage: 0 B of 16 EiB
          Streams: 0 of Unlimited
        Consumers: 0 of Unlimited

```

``` BASH
# 在云端
$ kubectl apply -f - << EOF
apiVersion: jetstream.nats.io/v1beta1
kind: Stream
metadata:
  name: edge01
spec:
  name: edge01
  subjects: ["edge01.*"]
  storage: memory
  maxAge: 1h
  replicas: 1
EOF
$ kubectl apply -f - << EOF
apiVersion: jetstream.nats.io/v1beta1
kind: Consumer
metadata:
  name: edge01
spec:
  streamName: edge01
  durableName: edge01
  deliverPolicy: all
  filterSubject: edge01.received
  maxDeliver: 20
  ackPolicy: explicit
EOF

$ kubectl exec -it nats-cloud-0 -- sh
$ nats -s nats://ce1:ce1@127.0.0.1:4222 str ls
Streams:

	edge01

$ nats -s nats://ce1:ce1@127.0.0.1:4222 con ls edge01
No Consumers defined
$ nats -s nats://ce1:ce1@127.0.0.1:4222 con ls edge01
Consumers for Stream edge01:

	edge01

# 在边缘端
$ nats -s nats://s1:s1@127.0.0.1:4222 str ls
Streams:

	edge01

$ nats -s nats://s1:s1@127.0.0.1:4222 con ls edge01
No Consumers defined
$ nats -s nats://s1:s1@127.0.0.1:4222 con ls edge01
Consumers for Stream edge01:

	edge01

```

**消息验证**

``` BASH
# 在边缘端
$ nats -s nats://s1:s1@127.0.0.1:4222 request edge01.received "hello from isol
ate edge"
11:51:56 Sending request on "edge01.received"
11:51:56 Received on "_INBOX.XNh702uWHRQJR0mYTosFak.rbbBtSiS" rtt 1.02546ms
{"stream":"edge01","seq":1}

# 在云端
$ nats -s nats://ce1:ce1@127.0.0.1:4222 con next edge01 edge01
[11:52:48] subj: edge01.received / tries: 1 / cons seq: 1 / str seq: 1 / pending: 0

hello from isolate edge

Acknowledged message

# 云端 ping 边缘无法连接
$ ping 192.168.122.116
PING 192.168.122.116 (192.168.122.116): 56 data bytes
^C
--- 192.168.122.116 ping statistics ---
7 packets transmitted, 0 packets received, 100% packet loss
```
