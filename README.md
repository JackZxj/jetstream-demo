``` bash
$ docker build -t 172.31.0.7:5000/js-demo:v0.5 .
$ docker push 172.31.0.7:5000/js-demo:v0.5
$ kubectl apply -f deployment.yaml
$ kubectl delete -f deployment.yaml
```