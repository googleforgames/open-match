This folder provides a minimal Dockerfile and k8s resource definition to build a sample director for Open Match tutorials.

Run the command below to define a variable for your personal container registry in your current terminal session
```
# Specify your Registry URL here.
REGISTRY=[YOUR_REGISTRY_URL]
```

To build your director image, run:
```
docker build -t $REGISTRY/om-director .
```

To push your director image, run:
```
docker push $REGISTRY/om-director
```

To deploy your Docker image into your Kubernetes cluster, run:
```
sed "s|registry_placeholder|$REGISTRY|g" om-director.yaml | kubectl apply -f -
```
