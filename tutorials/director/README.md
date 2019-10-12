This folder provides a minimal Dockerfile to build a sample director for Open Match tutorials

To build your director image, run:
```
docker build -t $(YOUR_PERSONAL_REGISTRY)/om-director .
```

To push your director image, run:
```
docker push $(YOUR_PERSONAL_REGISTRY)/om-director
```

To deploy your Docker image into your Kubernetes cluster, run:
```
kubectl apply -f om-director.yaml
```
