``` bash
$ docker build -t 172.31.0.7:5000/js-demo:v0.2 .
$ docker push 172.31.0.7:5000/js-demo:v0.2
$ kubectl apply -f deployment.yaml
$ kubectl delete -f deployment.yaml
```