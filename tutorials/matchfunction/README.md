This folder provides a minimal Dockerfile to build a sample match function for Open Match tutorials

To build your director image, run:
```
docker build -t $(YOUR_PERSONAL_REGISTRY)/om-function .
```

To push your director image, run:
```
docker push $(YOUR_PERSONAL_REGISTRY)/om-function
```

To deploy your Docker image into your Kubernetes cluster, run:
```
kubectl apply -f om-function.yaml
```
