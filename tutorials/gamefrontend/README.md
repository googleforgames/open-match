This folder provides a minimal Dockerfile and k8s resource definition to build a sample game-frontend for Open Match tutorials.

Run the command below to define a variable for your personal container registry in your current terminal session
```
# Specify your Registry URL here.
REGISTRY=[YOUR_REGISTRY_URL]
```

To build your game-frontend image, run:
```
docker build -t $REGISTRY/om-game-frontend .
```

To push your game-frontend image, run:
```
docker push $REGISTRY/om-game-frontend
```

To deploy your Docker image into your Kubernetes cluster, run:
```
sed "s|registry_placeholder|$REGISTRY|g" om-game-frontend.yaml | kubectl apply -f -
```
