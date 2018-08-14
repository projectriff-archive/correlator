
## build

```
./build.sh
```

## deploy

```
./deploy.sh
```

## invoke

to send to the "numbers" channel, where downstream it eventually replies
to the "replies" channel:

```
curl http://$(minikube ip):32380/numbers -H 'Host: correlator.default.example.com' -H 'Content-Type: text/plain' -d 7
```

