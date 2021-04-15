# Quick start NATS 2.2 with kind

About `kind` :

> kind is a tool for running local Kubernetes clusters using Docker container “nodes”.
> kind was primarily designed for testing Kubernetes itself, but may be used for local development or CI.

Install `kind` : https://kind.sigs.k8s.io/docs/user/quick-start/

## Prepare

``` BASH
# prepare a three nodes cluster
$ cat > kind-config.yaml << EOF
# three node (two workers) cluster config
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
# create the cluster
$ kind create cluster --name nats --config kind-config.yaml
 ✓ Ensuring node image (kindest/node:v1.20.2) 🖼
 ✓ Preparing nodes 📦 📦 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
Set kubectl context to "kind-nats"
You can now use your cluster with:

kubectl cluster-info --context kind-nats

Not sure what to do next? 😅  Check out https://kind.sigs.k8s.io/docs/user/quick-start/

# load images
$ docker pull natsio/nats-box:latest
$ kind load docker-image natsio/nats-box:latest --name nats
$ docker pull synadia/nats-server:nightly-20210412
$ kind load docker-image synadia/nats-server:nightly-20210412 --name nats

$ kubectl get all -A --context kind-nats

# get current context
$ kubectl config current-context
# switch current context to kind-nats
$ kubectl config use-context kind-nats
```

## Start

``` bash
# run a nfs server
$ docker run -d --name nfs --privileged -v $(pwd)/nats-accounts:/accounts -e SHARED_DIRECTORY=/accounts --network kind itsthenetwork/nfs-server-alpine:latest

###############################################################
# If you have a complete k8s cluster, you can start form here #
###############################################################

# prepare NATS system account
$ kubectl apply -f nsc-init.yaml
# you can get logs like this:
$ kubectl logs nsc-init-shr9z
[ OK ] generated and stored operator key "ODADDEDJHRMLXRVMX4YLFRWEAKPGFXRAUX3V4PPNUPPB5M4JRO2KEOXG"
[ OK ] added operator "inspur"
[ OK ] When running your own nats-server, make sure they run at least version 2.2.0
[ OK ] generated and stored account key "ABZOQRPOIXQYFQXXZM5RHMK6XKWRV2VX5GRBS5JEKVN4NCYY6NO7TSMN"
[ OK ] added account "SYS"
[ OK ] generated and stored user key "UBXS7YFBB7HYCUQPK6TQE3FKWBJQPMKPT6H57ANS52KJ5TRRBDE4IL4R"
[ OK ] generated user creds file `/nsc/nkeys/creds/inspur/SYS/sys.creds`
[ OK ] added user "sys" to account "SYS"
[ OK ] strict signing key usage set to: false
[ OK ] set system account "ABZOQRPOIXQYFQXXZM5RHMK6XKWRV2VX5GRBS5JEKVN4NCYY6NO7TSMN"
[ OK ] edited operator "inspur"

# start NATS cloud cluster
# NOTE: in fact, the nodes should use different resolver path
$ kubectl apply -f nats-cloud.yaml
# wait for one of nats-cloud ready, push the system account to NATS cluster
$ kubectl apply -f nsc-2init.yaml
# you can get logs like this:
$ kubectl logs nsc-2init-5twh5
[ OK ] strict signing key usage set to: false
[ OK ] set account jwt server url to "nats://nats-cloud.default.svc.cluster.local:4222"
[ OK ] added service url "nats://nats-cloud.default.svc.cluster.local:4222"
[ OK ] edited operator "inspur"
[ OK ] push to nats-server "nats://nats-cloud.default.svc.cluster.local:4222" using system account "SYS" user "sys":
       [ OK ] push SYS to nats-server with nats account resolver:
              [ OK ] pushed "SYS" to nats-server nats-cloud-1: jwt updated
              [ OK ] pushed "SYS" to nats-server nats-cloud-0: jwt updated
              [ OK ] pushed to a total of 2 nats-server


# create a account for jetstream leafnode
$ kubectl apply -f nsc-edge.yaml
# you can get logs like this:
$ kubectl logs nsc-edge-kwqrz
[ OK ] generated and stored account key "AB6MAVUM62FFSJ73HHCJM6KY2JOCAEKW6O6YDHTEEHRJCP7OA7V5UZQ2"
[ OK ] added account "A"
[ OK ] generated and stored user key "UAFBVDKH2KU4IXLT5KTOCQGTTD3NPLH47TJTJATQBJCXNEROTYREKAEE"
[ OK ] generated user creds file `/nsc/nkeys/creds/inspur/A/a.creds`
[ OK ] added user "a" to account "A"
[ OK ] push to nats-server "nats://nats-cloud.default.svc.cluster.local:4222" using system account "SYS" user "sys":
       [ OK ] push A to nats-server with nats account resolver:
              [ OK ] pushed "A" to nats-server nats-cloud-2: jwt updated
              [ OK ] pushed "A" to nats-server nats-cloud-0: jwt updated
              [ OK ] pushed to a total of 2 nats-server


# create a configmap for jetstream leafnode. or you can use secret instead
$ kubectl create configmap nats-edge-user-a --from-file=a.creds=$(pwd)/nats-accounts/nkeys/creds/inspur/A/a.creds
kubectl create configmap nats-edge-user-b --from-file=b.creds=$(pwd)/nats-accounts/nkeys/creds/inspur/B/b.creds
# start a jetstream leafnode
$ kubectl apply -f nats-edge.yaml
```

## Testing

### normal message test

``` bash
########## cloud to edge ##########
# terminal 1
$ kubectl exec -it nats-edge-0 -- sh
$ nats sub test
[#1] Received on "test"
hello

# terminal 2
$ kubectl exec -it nats-cloud-0 -- sh
$ nats --creds /nsc/nkeys/creds/inspur/A/a.creds pub test hello
03:05:29 Published 5 bytes to "test"

########## edge to cloud ##########
# terminal 2
$ nats --creds /nsc/nkeys/creds/inspur/A/a.creds sub test
03:06:25 Subscribing on test
[#1] Received on "test"
hello

# terminal 1
$ nats pub test hello
03:06:36 Published 5 bytes to "test"
```

### jetstream test

``` BASH
# terminal 2
$ kubectl exec -it nats-edge-0 -- sh
# create stream
$ nats str add mysqldb1 --subjects="mysqldb1.>" --storage=file --retention=workq --discard=new --max-msgs=-1 --max-bytes=-1 --max-age=-1 --max-msg-size=-1 --dupe-window=2s --replicas=1
# create consumer
$ nats con add mysqldb1 mysqldb1 --deliver=all --replay=instant --filter="mysqldb1.1" --max-deliver=-1 --max-pending=1 --pull
# Sending
$ nats request mysqldb1.1 hello1
06:48:23 Sending request on "mysqldb1.1"
06:48:23 Received on "_INBOX.daVT6zJqxXHX4wJrAZpanP.6V8mRDDE" rtt 596.559µs
{"stream":"mysqldb1","seq":1}
# 
# terminal 1
$ kubectl exec -it nats-cloud-0 -- sh
# recieve with creds
$ nats --creds /nsc/nkeys/creds/inspur/A/a.creds con next mysqldb1 mysqldb1
[06:48:23] subj: mysqldb1.1 / tries: 1 / cons seq: 1 / str seq: 1 / pending: 0

hello1

Acknowledged message
```

### controller test 

``` BASH
# this image was built by myself, you can use the official image.
$ kind load docker-image natsio/jetstream-controller:20210413 --name nats
$ kubectl apply -f nack-crds.yml
$ kubectl apply -f nack-rbac.yml
$ kubectl apply -f nack-controller.yaml

# create stream & consumer
$ kubectl apply -f sampleStr.yaml
$ kubectl apply -f sampleCon.yaml

# test the stream and consumer
# terminal 2
$ kubectl exec -it nats-edge-0 -- sh
# Sending
$ nats request samplestr.samplecon hello1
02:40:15 Sending request on "samplestr.samplecon"
02:40:15 Received on "_INBOX.2tRs6XN4TvVxNXcddR7K51.BRb8HbKl" rtt 534.926µs
{"stream":"samplestr","seq":1}
# 
# terminal 1
$ kubectl exec -it nats-cloud-0 -- sh
# recieve with creds
$ nats --creds /nsc/nkeys/creds/inspur/A/a.creds con next samplestr samplecon
[02:40:19] subj: samplestr.samplecon / tries: 1 / cons seq: 1 / str seq: 1 / pending: 0

hello1

Acknowledged message

# 
# remove

```

## Remove

``` bash
$ kubectl delete -f sampleCon.yaml
$ kubectl delete -f sampleStr.yaml

$ kubectl delete -f nack-controller.yaml
$ kubectl delete -f nack-rbac.yml
$ kubectl delete -f nack-crds.yml

$ kubectl delete -f nats-edge.yaml
$ kubectl delete configmap nats-edge-user-a
$ kubectl delete -f nsc-edge.yaml

$ kubectl delete -f nsc-2init.yaml
$ kubectl delete -f nats-cloud.yaml
$ kubectl delete -f nsc-init.yaml

$ rm -rf $(pwd)/nats-accounts/*

# switch to default context
# you can use `kubectl config view` to get contexts in kubernetes' config
$ kubectl config use-context kubernetes-admin@kubernetes
$ kind delete cluster --name nats
```