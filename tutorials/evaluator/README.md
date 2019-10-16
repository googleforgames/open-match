This folder provides a minimal Dockerfile and k8s resource definition to build a sample evaluator for Open Match tutorials.

Run the command below to define a variable for your personal container registry in your current terminal session
```
# Example - replace open-match-public-images with your personal registry ID
YOUR_PERSONAL_REGISTRY=gcr.io/open-match-public-images
```

To build your evaluator image, run:
```
docker build -t $YOUR_PERSONAL_REGISTRY/om-evaluator .
```

To push your evaluator image, run:
```
docker push $YOUR_PERSONAL_REGISTRY/om-evaluator
```

To deploy your Docker image into your Kubernetes cluster, run:
```
sed "s|registry_placeholder|$YOUR_PERSONAL_REGISTRY|g" om-evaluator.yaml | kubectl apply -f -
```
