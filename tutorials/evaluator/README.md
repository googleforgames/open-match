This folder provides a minimal Dockerfile and k8s resource definition to build a sample evaluator for Open Match tutorials.

Run the command below to define a variable for your personal container registry in your current terminal session
```
# Specify your Registry URL here.
REGISTRY=[YOUR_REGISTRY_URL]
```

To build your evaluator image, run:
```
docker build -t $REGISTRY/om-evaluator .
```

To push your evaluator image, run:
```
docker push $REGISTRY/om-evaluator
```

To deploy your Docker image into your Kubernetes cluster, run:
```
sed "s|registry_placeholder|$REGISTRY|g" om-evaluator.yaml | kubectl apply -f -
```
