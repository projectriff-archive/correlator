create functions:

```
riff function create node square \
  --git-repo https://github.com/trisberg/node-fun-square.git \
  --artifact square.js \
  --image markfisher/demo-node-square

riff function create command hello \
  --git-repo https://github.com/markfisher/riff-sample-hello.git \
  --artifact hello.sh \
  --image markfisher/demo-command-hello
```

create channels:

```
riff channel create numbers --cluster-bus stub
riff channel create squares --cluster-bus stub
riff channel create replies --cluster-bus stub
```

build and deploy the correlator:

```
docker build . -t dev.local/correlator:0.0.1
riff service create correlator --image dev.local/correlator:0.0.1
```

create subscriptions:

```
riff service subscribe square --input numbers --output squares
riff service subscribe hello --input squares --output replies
riff service subscribe correlator --input replies
```

invoke with a blocking request:

```
curl http://$(minikube ip):32380/numbers \
  -HHost:correlator.default.example.com \
  -HContent-Type:text/plain \
  -Hknative-blocking-request:true -w'\n' \
  -d 7
```

you should see:

```
hello 49
```

invoke with a non-blocking request and log the correlation ID:

```
curl http://$(minikube ip):32380/numbers -is \
  -HHost:correlator.default.example.com \
  -HContent-Type:text/plain -w'\n' \
  -d 11 \
  | grep knative-correlation-id \
  | sed "s/.*knative-correlation-id: \(.*\)/\1/g"
```

you should see a correlation ID similar to this:

```
b9962c86-b01b-4bc5-b624-c417032d1e4b
```

retrieve the result for the returned correlation ID:

```
curl http://$(minikube ip):32380/b9962c86-b01b-4bc5-b624-c417032d1e4b \
  -HHost:correlator.default.example.com -w'\n'
```

you should see:

```
hello 121
```
